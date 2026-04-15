package main

// enricher fetches device metadata (name, location) from the PostgreSQL
// device registry and caches results locally with a TTL.
//
// Why cache? The device registry changes rarely (new devices registered,
// firmware updated). Hitting PostgreSQL for every metric at 1000 msg/sec
// would put unnecessary load on the DB. A 5-minute TTL means stale data
// is at most 5 minutes old — acceptable for enrichment.
//
// Cache implementation: simple map with expiry timestamps. A background
// goroutine is NOT used — entries are evicted lazily on read (check expiry,
// re-fetch if stale). This avoids goroutine management complexity.
//
// On cache miss (first access or TTL expired): fetch from PostgreSQL via pgxpool.
// On PostgreSQL error: return the stale cached entry if available, log a warning.
// If no cached entry exists and PostgreSQL is down: return an error to the caller.
type enricher struct {
	// TODO: *pgxpool.Pool, map[deviceID]cacheEntry, TTL duration
}

// cacheEntry wraps a Device with its expiry time.
type cacheEntry struct {
	// TODO: *models.Device, expiresAt time.Time
}

// newEnricher creates an enricher backed by the given PostgreSQL pool.
func newEnricher() (*enricher, error) {
	panic("not implemented — implement in Phase 3")
}

// GetDevice returns the Device for the given ID, using the cache.
// Fetches from PostgreSQL on cache miss or TTL expiry.
func (e *enricher) GetDevice() error {
	panic("not implemented — implement in Phase 3")
}
