// Package kafka provides shared Kafka producer and consumer wrappers
// built on top of franz-go (github.com/twmb/franz-go).
//
// All producers are idempotent by default (ADR-003).
// All consumers use manual offset commits (ADR-002).
// Unprocessable messages are routed to a DLQ topic (ADR-007).
package kafka

// Topic constants — never hardcode topic names in service code (see CLAUDE.md).
const (
	TopicDeviceMetrics    = "device.metrics"
	TopicDeviceAggregated = "device.aggregated"
	TopicDeviceAlerts     = "device.alerts"
	TopicMetricsDLQ       = "device.metrics.dlq"
	TopicAlertsDLQ        = "device.alerts.dlq"
)

// Consumer group identifiers — one group per service.
// All instances of the same service share the same group ID so Kafka
// distributes partitions across them automatically.
const (
	GroupIngestor  = "telemetryflow-ingestor"
	GroupProcessor = "telemetryflow-processor"
	GroupSink      = "telemetryflow-sink"
)
