package main

// writer handles batched inserts of DeviceMetric records into the
// TimescaleDB device_metrics hypertable.
//
// Insert strategy: COPY protocol via pgx's CopyFrom, which is significantly
// faster than repeated INSERT statements for bulk loads. At 1000 devices/sec,
// batching 500 records every 500ms means ~2 COPY calls/sec instead of 1000
// INSERT calls/sec.
//
// Exactly-once guarantee (ADR-004): the write and offset commit happen inside
// a Kafka transaction. The writer is given the transactional producer to call
// CommitTransaction after the DB write succeeds. If the DB write fails, the
// caller aborts the transaction and does not commit offsets — the records will
// be reprocessed on the next poll.
type writer struct {
	// TODO: *pgxpool.Pool (TimescaleDB)
}

// newWriter creates a writer connected to the given TimescaleDB pool.
func newWriter() (*writer, error) {
	panic("not implemented — implement in Phase 2")
}

// WriteBatch inserts a slice of DeviceMetric records into device_metrics
// using the pgx CopyFrom protocol. Wraps errors with context.
func (w *writer) WriteBatch() error {
	panic("not implemented — implement in Phase 2")
}
