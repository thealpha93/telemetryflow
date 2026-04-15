package kafka

// Producer is an idempotent franz-go producer wrapper.
//
// Every producer in TelemetryFlow is idempotent (ADR-003):
//   - RequiredAcks = AllISRAcks  (acks=all — leader + all in-sync replicas confirm)
//   - Idempotency enabled        (exactly-once delivery at the producer level)
//   - LZ4 batch compression      (reduces network bytes to Confluent Cloud)
//   - 5ms linger                 (batches records before sending, improves throughput)
//   - Sticky key partitioning    (all records with the same key go to the same partition)
//
// Usage:
//
//	p, err := NewProducer(cfg.Kafka)
//	defer p.Close()
//	err = p.Produce(ctx, topic, key, value)
type Producer struct {
	// TODO: embed *kgo.Client
}

// NewProducer creates and returns a connected idempotent Producer.
func NewProducer() (*Producer, error) {
	panic("not implemented — implement in Phase 1")
}

// Produce sends a single record to the given topic synchronously.
// The record key determines which partition receives the message.
func (p *Producer) Produce() error {
	panic("not implemented — implement in Phase 1")
}

// Close flushes any buffered records and closes the underlying client.
// Always call Close in a defer or shutdown handler.
func (p *Producer) Close() {
	panic("not implemented — implement in Phase 1")
}
