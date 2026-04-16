package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/telemetryflow/config"
)

// NewTimescalePool creates and validates a pgxpool.Pool connected to TimescaleDB.
//
// TimescaleDB is PostgreSQL with the TimescaleDB extension, so the same pgx
// driver and pgxpool work unchanged. The only differences are:
//   - TLS required (sslmode=require in the DSN for Timescale Cloud)
//   - Hypertable-aware batch insert helpers (see WriteBatch)
//
// Call pool.Close() in the service's shutdown handler.
func NewTimescalePool(ctx context.Context, cfg config.TimescaleConfig) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: creating timescale pool: %w", err)
	}

	// Ping verifies TLS, credentials, and that the TimescaleDB extension is
	// accessible. A wrong sslmode or invalid cert fails here, not mid-pipeline.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: pinging timescale at %q: %w", cfg.DSN, err)
	}

	return pool, nil
}
