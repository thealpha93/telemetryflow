package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/telemetryflow/pkg/models"
)

// enricher fetches device metadata from the PostgreSQL device registry and
// caches results locally with a TTL (default 5 minutes).
//
// Why cache?
// At 1000 msg/sec, a PostgreSQL round-trip per message would mean 1000
// queries/sec for data that changes rarely. The cache reduces this to one
// query per device per TTL period — roughly 3-4 queries/hour for 1000 devices.
//
// Eviction strategy: lazy. Stale entries are detected and refreshed on access.
// No background goroutine needed — this avoids goroutine lifecycle complexity
// and means stale data is never served if the device is actively producing.
//
// Fault tolerance:
//   - Cache hit, not expired     → return cached value (fast path)
//   - Cache miss or TTL expired  → query PostgreSQL, update cache
//   - PostgreSQL error, stale entry exists → serve stale entry with a warning
//   - PostgreSQL error, no cache entry → return error to caller
type enricher struct {
	pool *pgxpool.Pool
	ttl  time.Duration

	mu    sync.RWMutex
	cache map[string]cacheEntry
}

// cacheEntry wraps a Device with its expiry timestamp.
type cacheEntry struct {
	device    models.Device
	expiresAt time.Time
}

// newEnricher creates an enricher backed by pool with the given TTL.
func newEnricher(pool *pgxpool.Pool, ttl time.Duration) *enricher {
	return &enricher{
		pool:  pool,
		ttl:   ttl,
		cache: make(map[string]cacheEntry),
	}
}

// GetDevice returns the Device for deviceID using the local cache.
// Falls back to PostgreSQL on cache miss or TTL expiry.
func (e *enricher) GetDevice(ctx context.Context, deviceID string) (models.Device, error) {
	// Fast path: valid cached entry.
	e.mu.RLock()
	entry, ok := e.cache[deviceID]
	e.mu.RUnlock()

	if ok && time.Now().Before(entry.expiresAt) {
		return entry.device, nil
	}

	// Cache miss or expired — query PostgreSQL.
	var d models.Device
	err := e.pool.QueryRow(ctx,
		`SELECT device_id, name, location_id, firmware_version
		 FROM devices WHERE device_id = $1`,
		deviceID,
	).Scan(&d.DeviceID, &d.Name, &d.LocationID, &d.FirmwareVersion)

	if err != nil {
		if ok {
			// Serve the stale entry rather than propagating a transient DB error.
			// The enrichment data is used for metadata only — slightly stale is fine.
			slog.Warn("enricher: postgres error, serving stale cache entry",
				"device_id", deviceID,
				"error", err,
				"stale_until", entry.expiresAt,
			)
			return entry.device, nil
		}
		return models.Device{}, fmt.Errorf("enricher: fetching device %q: %w", deviceID, err)
	}

	d.Active = true

	e.mu.Lock()
	e.cache[deviceID] = cacheEntry{
		device:    d,
		expiresAt: time.Now().Add(e.ttl),
	}
	e.mu.Unlock()

	return d, nil
}
