package main

// sinkConsumer fans records from two topics into two separate writer pipelines.
//
// Both device.aggregated and device.alerts are consumed in the same consumer
// group. The consumer inspects the topic field on each record and routes it:
//
//	device.aggregated → aggregatedWriter.WriteBatch()
//	device.alerts     → alertWriter.WriteBatch()
//
// Batching: each topic's records accumulate independently. A flush is triggered
// when either batch reaches SINK_BATCH_SIZE OR SINK_FLUSH_INTERVAL_MS elapses.
// The ticker + select pattern (not time.Sleep) is used for the interval flush.
//
// Offset commit happens per-record after confirmed write. This gives
// at-least-once delivery — on replay, the idempotent upserts handle duplicates.
type sinkConsumer struct {
	// TODO: *kafka.Consumer, *aggregatedWriter, *alertWriter, *kafka.DLQPublisher
}

// newSinkConsumer wires up the consumer with its writers and DLQ publisher.
func newSinkConsumer() (*sinkConsumer, error) {
	panic("not implemented — implement in Phase 4")
}

// Run starts the consume-route-flush loop. Blocks until ctx is cancelled.
func (c *sinkConsumer) Run() error {
	panic("not implemented — implement in Phase 4")
}

// Shutdown flushes all pending batches and commits offsets.
func (c *sinkConsumer) Shutdown() error {
	panic("not implemented — implement in Phase 4")
}
