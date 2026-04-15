package main

// aggregatedWriter persists window aggregates to the TimescaleDB
// aggregated_metrics table using idempotent upserts.
//
// Upsert key: (device_id, bucket) — one row per device per time bucket.
// ON CONFLICT DO UPDATE ensures replayed records update rather than error.
type aggregatedWriter struct {
	// TODO: *pgxpool.Pool (TimescaleDB)
}

// newAggregatedWriter creates a writer for the TimescaleDB aggregated table.
func newAggregatedWriter() (*aggregatedWriter, error) {
	panic("not implemented — implement in Phase 4")
}

// WriteBatch upserts a slice of aggregated metrics records.
func (w *aggregatedWriter) WriteBatch() error {
	panic("not implemented — implement in Phase 4")
}

// alertWriter persists alerts to the PostgreSQL alerts and alert_events tables.
//
// Write pattern per alert:
//  1. Upsert into alerts (ON CONFLICT on alert_id DO UPDATE) — current state
//  2. Insert into alert_events (always append) — full audit trail
//
// Both writes happen in a single PostgreSQL transaction so they succeed or
// fail together. A partial write (alert row without event row) is never possible.
type alertWriter struct {
	// TODO: *pgxpool.Pool (PostgreSQL)
}

// newAlertWriter creates a writer for the PostgreSQL alerts tables.
func newAlertWriter() (*alertWriter, error) {
	panic("not implemented — implement in Phase 4")
}

// WriteBatch upserts a slice of alerts and appends their corresponding events.
func (w *alertWriter) WriteBatch() error {
	panic("not implemented — implement in Phase 4")
}
