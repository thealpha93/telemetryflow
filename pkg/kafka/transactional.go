package kafka

// TransactionalProducer wraps franz-go's transactional producer API to provide
// exactly-once semantics (EOS) for the ingestor service (ADR-004).
//
// EOS in Kafka means: a message is written to the output topic if and only if
// the consumer offset is also committed — both happen atomically in the same
// Kafka transaction. This prevents duplicates even if the process crashes
// between writing and committing.
//
// Only the ingestor uses this. Other services use idempotent producers +
// database-level ON CONFLICT upserts, which is sufficient and cheaper (ADR-004).
//
// EOS requires:
//   - A unique TransactionalID per producer instance
//   - InitProducerID handshake on startup
//   - BeginTransaction / CommitTransaction / AbortTransaction calls around each batch
//
// Usage:
//
//	tp, err := NewTransactionalProducer(cfg.Kafka, "ingestor-0")
//	defer tp.Close()
//
//	tp.BeginTransaction()
//	tp.Produce(ctx, topic, key, value)
//	tp.CommitTransaction(ctx)  // or AbortTransaction on error
type TransactionalProducer struct {
	// TODO: embed *kgo.Client with transactional options
}

// NewTransactionalProducer creates a producer with a fixed transactional ID.
// The transactional ID must be unique per producer instance and stable across
// restarts (use the partition number or pod name as a suffix).
func NewTransactionalProducer() (*TransactionalProducer, error) {
	panic("not implemented — implement in Phase 1")
}

// BeginTransaction starts a new Kafka transaction.
func (tp *TransactionalProducer) BeginTransaction() error {
	panic("not implemented — implement in Phase 1")
}

// CommitTransaction atomically commits the transaction and the consumer offsets.
func (tp *TransactionalProducer) CommitTransaction() error {
	panic("not implemented — implement in Phase 1")
}

// AbortTransaction rolls back the current transaction.
// Call this on any processing error to discard the in-flight batch.
func (tp *TransactionalProducer) AbortTransaction() error {
	panic("not implemented — implement in Phase 1")
}

// Close flushes and closes the transactional producer.
func (tp *TransactionalProducer) Close() {
	panic("not implemented — implement in Phase 1")
}
