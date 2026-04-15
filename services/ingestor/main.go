// Ingestor consumes raw device metrics from device.metrics and writes them
// to the TimescaleDB hypertable using batched, exactly-once inserts (ADR-004, ADR-005).
//
// Key behaviours:
//   - Exactly-once semantics via Kafka transactions: DB write + offset commit
//     are atomic. A crash mid-batch causes reprocessing with no duplicates.
//   - Batched TimescaleDB writes: accumulates up to INGESTOR_BATCH_SIZE records
//     or INGESTOR_FLUSH_INTERVAL_MS ms, whichever comes first (ADR-005).
//   - DLQ: records that fail deserialization or DB write after 3 retries are
//     published to device.metrics.dlq with error headers (ADR-007).
//   - Per-partition processing: each Kafka partition is processed by at most
//     one ingestor instance at a time (guaranteed by consumer groups).
//
// Scaling: run multiple ingestor instances with the same consumer group.
// Kafka distributes the 12 partitions across instances automatically.
//
// Configuration (env vars):
//
//	INGESTOR_BATCH_SIZE        records per flush (default: 500)
//	INGESTOR_FLUSH_INTERVAL_MS max ms between flushes (default: 500)
//	INGESTOR_CONSUMER_GROUP    Kafka consumer group ID
package main

func main() {
	panic("not implemented — implement in Phase 2")
}
