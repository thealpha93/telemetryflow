package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// RunMigrations applies all pending goose migrations from dir against the
// database backing pool.
//
// goose tracks applied migrations in a goose_db_version table — running this
// twice is safe; already-applied migrations are skipped.
//
// dir must contain numbered .sql files with goose annotations:
//
//	-- +goose Up
//	CREATE TABLE ...;
//
//	-- +goose Down
//	DROP TABLE ...;
//
// Call once per service on startup before accepting any traffic. If migrations
// fail, the service should exit — operating on a stale schema is worse than
// not starting.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	// stdlib.OpenDBFromPool wraps the pgxpool in a database/sql adapter.
	// goose uses database/sql, so this is the bridge between the two APIs.
	// The underlying connections still come from the pool.
	sqlDB := stdlib.OpenDBFromPool(pool)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("db: setting goose dialect: %w", err)
	}

	if err := goose.UpContext(ctx, sqlDB, dir); err != nil {
		return fmt.Errorf("db: running migrations from %q: %w", dir, err)
	}

	return nil
}
