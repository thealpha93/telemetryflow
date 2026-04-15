package db

// RunMigrations applies all pending goose migrations from the given directory
// against the given database connection.
//
// Goose tracks applied migrations in a goose_db_version table. Running
// migrations twice is safe — already-applied migrations are skipped.
//
// Migration files use the naming convention:
//
//	NNN_description.sql  (e.g. 001_create_devices.sql)
//
// Each file contains goose annotations:
//
//	-- +goose Up
//	CREATE TABLE ...;
//
//	-- +goose Down
//	DROP TABLE ...;
//
// Usage (called from each service's main.go on startup):
//
//	if err := db.RunMigrations(ctx, pool, "migrations/postgres"); err != nil {
//	    return fmt.Errorf("postgres migrations failed: %w", err)
//	}
func RunMigrations() error {
	panic("not implemented — implement in Phase 1")
}
