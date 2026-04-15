// Processor is the stream processing service. It consumes device.metrics,
// maintains per-device sliding windows, detects anomalies, enriches events
// with device registry data, and produces to device.aggregated and device.alerts.
//
// Key behaviours:
//   - Sliding window aggregations: tracks the last N seconds of readings
//     per device in memory. Windows are keyed by device_id, and because
//     Kafka partitions by device_id, all readings for a device always land
//     on the same processor instance — no cross-instance coordination needed.
//   - Anomaly detection: evaluates temperature high/low and battery low
//     thresholds on each window. On breach, emits a DeviceAlert to device.alerts.
//   - Enrichment: augments aggregates with device name and location from
//     the PostgreSQL device registry, cached locally with a TTL (ADR-006).
//   - Cooperative rebalancing: when a new processor instance joins the group,
//     only the reassigned partitions pause — other partitions keep flowing.
//     The in-memory window state for reassigned partitions is discarded;
//     new state rebuilds from the next batch of records (acceptable trade-off).
//
// Scaling: run up to 12 instances (one per partition). Kafka distributes
// partitions using the cooperative sticky assignor (franz-go default).
//
// Configuration (env vars):
//
//	PROCESSOR_CONSUMER_GROUP              Kafka consumer group ID
//	PROCESSOR_WINDOW_SIZE_SECONDS         sliding window length (default: 60)
//	PROCESSOR_ANOMALY_TEMP_HIGH           °C threshold for TEMP_HIGH (default: 85.0)
//	PROCESSOR_ANOMALY_TEMP_LOW            °C threshold for TEMP_LOW  (default: -10.0)
//	PROCESSOR_ANOMALY_BATTERY_LOW         % threshold for BATTERY_LOW (default: 10.0)
//	PROCESSOR_ENRICHMENT_CACHE_TTL_SECONDS device cache TTL in seconds (default: 300)
package main

func main() {
	panic("not implemented — implement in Phase 3")
}
