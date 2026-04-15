package main

// processorConsumer drives the poll → process → publish loop.
//
// For each polled record:
//  1. Deserialize Avro → DeviceMetric
//  2. Push metric into the device's sliding window
//  3. Compute window aggregates (avg/max/min temp, avg battery, sample count)
//  4. Run anomaly detection rules on the window
//  5. Enrich aggregates with device metadata (name, location) from cache
//  6. Publish aggregates to device.aggregated
//  7. If anomaly detected: publish alert to device.alerts
//  8. Commit the offset — only after all publishes succeed
//
// If any step fails after retries, the record goes to device.metrics.dlq
// and the offset is committed to avoid blocking the partition.
type processorConsumer struct {
	// TODO: *kafka.Consumer, *windowStore, *anomalyDetector, *enricher, *publisher, *kafka.DLQPublisher
}

// newProcessorConsumer wires up the consumer with all its dependencies.
func newProcessorConsumer() (*processorConsumer, error) {
	panic("not implemented — implement in Phase 3")
}

// Run starts the consume-process-publish loop. Blocks until ctx is cancelled.
func (c *processorConsumer) Run() error {
	panic("not implemented — implement in Phase 3")
}

// Shutdown flushes any in-flight publishes before returning.
func (c *processorConsumer) Shutdown() error {
	panic("not implemented — implement in Phase 3")
}
