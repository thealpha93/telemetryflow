// Sink persists processed data from two topics to two databases:
//
//	device.aggregated → TimescaleDB (aggregated_metrics table)
//	device.alerts     → PostgreSQL  (alerts + alert_events tables)
//
// Key behaviours:
//   - Idempotent writes: all inserts use ON CONFLICT DO UPDATE (upsert).
//     If the sink crashes and replays a batch, duplicate records are silently
//     merged — no duplicate rows, no errors (ADR-004).
//   - Per-partition error isolation: a failure writing one partition's batch
//     does not block other partitions. Each partition's records are processed
//     and committed independently.
//   - Alert lifecycle: when an alert arrives, the sink writes to both the
//     alerts table (current state) and alert_events table (full audit trail).
//     This gives operators both a current-state view and a complete history.
//   - DLQ: records that fail after 3 retries go to device.alerts.dlq.
//
// Scaling: run multiple sink instances. Kafka distributes partitions across
// them using the consumer group mechanism.
//
// Configuration (env vars):
//
//	SINK_CONSUMER_GROUP    Kafka consumer group ID
//	SINK_BATCH_SIZE        records per flush (default: 100)
//	SINK_FLUSH_INTERVAL_MS max ms between flushes (default: 1000)
package main

func main() {
	panic("not implemented — implement in Phase 4")
}
