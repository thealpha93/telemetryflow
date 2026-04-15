// Package db provides database connection factories and helpers for
// TelemetryFlow's two databases:
//   - PostgreSQL (local): device registry, alerts, DLQ
//   - TimescaleDB (cloud): raw metrics hypertable, continuous aggregates
//
// All connections use pgxpool.Pool (ADR-009). Never use pgx.Conn directly —
// it serializes all operations on a single connection and panics under
// concurrent access.
package db

// NewPostgresPool creates and validates a pgxpool.Pool connected to PostgreSQL.
//
// The pool is configured from the DSN in cfg.Postgres.DSN.
// Pool size is capped at a sensible default; override via env var if needed.
//
// Call pool.Close() in the service's shutdown handler.
//
// Usage:
//
//	pool, err := db.NewPostgresPool(ctx, cfg.Postgres)
//	defer pool.Close()
func NewPostgresPool() error {
	panic("not implemented — implement in Phase 1")
}
