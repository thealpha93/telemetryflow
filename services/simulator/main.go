// Simulator is the device metric producer service.
//
// It models N IoT devices as concurrent goroutines, each maintaining its own
// state machine (normal → spike → drift → normal). On every tick it generates
// a realistic DeviceMetric reading and publishes it to the device.metrics
// Kafka topic as an Avro-encoded message keyed by device_id.
//
// Key behaviours:
//   - Each device runs in its own goroutine — no shared mutable state
//   - Messages are Avro-encoded using the Schema Registry wire format
//   - Producer is idempotent (ADR-003) — safe to retry on transient errors
//   - Chaos mode (SIMULATOR_CHAOS_MODE=true) randomly injects anomalies
//     to exercise the processor's anomaly detection
//   - Graceful shutdown: waits for all in-flight Produce calls to complete
//
// Configuration (env vars):
//
//	SIMULATOR_DEVICE_COUNT     number of simulated devices (default: 1000)
//	SIMULATOR_EMIT_INTERVAL_MS emit interval per device in ms (default: 1000)
//	SIMULATOR_CHAOS_MODE       inject random anomalies (default: false)
package main

func main() {
	panic("not implemented — implement in Phase 2")
}
