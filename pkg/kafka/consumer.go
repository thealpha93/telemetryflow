package kafka

// Consumer is a franz-go consumer group wrapper with manual offset commits.
//
// Key behaviours enforced here (ADR-002, ADR-006):
//   - AutoCommit is DISABLED — offsets are only committed after the caller
//     confirms successful processing. A crash before commit causes reprocessing,
//     which is correct at-least-once behaviour.
//   - BlockRebalanceOnPoll — prevents a rebalance from stealing partitions
//     mid-batch, which would leave in-flight records unprocessed.
//   - CooperativeStickyAssignor (franz-go default) — only moves partitions
//     that actually need to change owners, preserving in-memory window state
//     in the processor service (ADR-006).
//   - MaxPollRecords(500) — caps batch size per poll to bound memory usage.
//
// Usage:
//
//	c, err := NewConsumer(cfg.Kafka, GroupIngestor, TopicDeviceMetrics)
//	defer c.Close()
//
//	fetches := c.Poll(ctx)
//	for _, record := range fetches.Records() {
//	    err := process(record)
//	    if err != nil {
//	        dlq.Publish(ctx, record, err) // do NOT commit on failure
//	        continue
//	    }
//	    c.Commit(ctx, record) // commit only after confirmed processing
//	}
type Consumer struct {
	// TODO: embed *kgo.Client
}

// NewConsumer creates and returns a Consumer joined to the given group
// and subscribed to the given topics.
func NewConsumer() (*Consumer, error) {
	panic("not implemented — implement in Phase 1")
}

// Poll blocks until records are available or ctx is cancelled.
// Returns a batch of records from the subscribed topics.
func (c *Consumer) Poll() {
	panic("not implemented — implement in Phase 1")
}

// Commit marks the given record's offset as processed.
// Only call this after the record has been successfully written to the database.
func (c *Consumer) Commit() {
	panic("not implemented — implement in Phase 1")
}

// Close gracefully leaves the consumer group and closes the client.
func (c *Consumer) Close() {
	panic("not implemented — implement in Phase 1")
}
