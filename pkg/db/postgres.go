// Package db provides database connection factories for TelemetryFlow's
// two databases:
//   - PostgreSQL (local): device registry, alerts, DLQ
//   - TimescaleDB (cloud): raw metrics hypertable, continuous aggregates
//
// All connections use pgxpool.Pool (ADR-009). Never use pgx.Conn directly —
// it is not safe for concurrent use and serialises all operations.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/telemetryflow/config"
	"github.com/telemetryflow/pkg/models"
)

// NewPostgresPool creates, configures, and validates a pgxpool.Pool
// connected to the local PostgreSQL instance.
//
// The pool is sized by pgx defaults (max 4 connections) which is sufficient
// for our workload. Override via the DSN's pool_max_conns parameter if needed.
//
// Always call pool.Close() in the service's shutdown handler to release
// connections cleanly.
func NewPostgresPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: creating postgres pool: %w", err)
	}

	// Ping verifies the DSN is valid and the server is reachable.
	// Without this, a wrong password or hostname only surfaces on the
	// first query — potentially deep inside a request handler.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: pinging postgres at %q: %w", cfg.DSN, err)
	}

	return pool, nil
}

// LoadActiveDevices queries the device registry and returns all devices where
// active = true, ordered by device_id. This is the canonical way for services
// to discover which devices exist — the registry is the source of truth.
//
// Called by the simulator on startup to seed its goroutine pool, and by the
// processor's enricher to prime its local cache.
func LoadActiveDevices(ctx context.Context, pool *pgxpool.Pool) ([]models.Device, error) {
	const q = `
		SELECT device_id, name, location_id, firmware_version
		FROM devices
		WHERE active = true
		ORDER BY device_id`

	rows, err := pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("db: querying active devices: %w", err)
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var d models.Device
		if err := rows.Scan(&d.DeviceID, &d.Name, &d.LocationID, &d.FirmwareVersion); err != nil {
			return nil, fmt.Errorf("db: scanning device row: %w", err)
		}
		d.Active = true
		devices = append(devices, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: iterating device rows: %w", err)
	}

	return devices, nil
}
