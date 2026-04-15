package main

// publisher wraps the Kafka producers for the processor's two output topics:
//   - device.aggregated: window aggregates (avg/max/min per device per minute)
//   - device.alerts:     anomaly alerts (public contract — see CLAUDE.md)
//
// Both producers are idempotent (ADR-003). The processor does NOT use
// full EOS transactions (ADR-004) — the overhead is not justified here
// because the sink service uses ON CONFLICT upserts, making duplicate
// aggregates harmless.
//
// Publish order within a device: aggregated record is published first,
// then the alert (if any). Both are keyed by device_id so they land on
// the same partition in their respective topics — preserving per-device order.
type publisher struct {
	// TODO: *kafka.Producer (aggregated), *kafka.Producer (alerts)
	// TODO: *schema.Serializer (aggregated), *schema.Serializer (alerts)
}

// newPublisher creates a publisher connected to both output topics.
func newPublisher() (*publisher, error) {
	panic("not implemented — implement in Phase 3")
}

// PublishAggregate encodes and publishes a window aggregate to device.aggregated.
func (p *publisher) PublishAggregate() error {
	panic("not implemented — implement in Phase 3")
}

// PublishAlert encodes and publishes an alert to device.alerts.
func (p *publisher) PublishAlert() error {
	panic("not implemented — implement in Phase 3")
}

// Close flushes both producers.
func (p *publisher) Close() {
	panic("not implemented — implement in Phase 3")
}
