package db

// NewTimescalePool creates and validates a pgxpool.Pool connected to TimescaleDB.
//
// TimescaleDB is PostgreSQL with the TimescaleDB extension, so the same pgx
// driver and pgxpool work unchanged. The only differences are:
//   - TLS required (sslmode=require in the DSN for Timescale Cloud)
//   - Hypertable-aware batch insert helpers (see WriteBatch)
//
// Call pool.Close() in the service's shutdown handler.
//
// Usage:
//
//	pool, err := db.NewTimescalePool(ctx, cfg.Timescale)
//	defer pool.Close()
func NewTimescalePool() error {
	panic("not implemented — implement in Phase 1")
}
