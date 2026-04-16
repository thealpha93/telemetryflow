package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/telemetryflow/config"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
)

// TransactionalProducer wraps franz-go's transactional producer API for
// exactly-once semantics (EOS) within Kafka — ADR-004.
//
// When does EOS actually help?
// The processor reads device.metrics, transforms records, and writes to
// device.aggregated and device.alerts. If it crashes between writing to Kafka
// and committing the input offset, without transactions it would re-process
// the same input and produce duplicate output records. With transactions, the
// output write and the input offset commit happen atomically — either both
// land or neither does.
//
// Note: EOS is Kafka-to-Kafka only. It does NOT extend to external systems
// like PostgreSQL. The ingestor writes to TimescaleDB, so it uses at-least-once
// + idempotent DB upserts instead of transactions (also ADR-004).
//
// Transaction lifecycle per batch:
//
//	tp.BeginTransaction()
//	for _, record := range batch {
//	    tp.Produce(ctx, topic, key, value)
//	}
//	if processingSucceeded {
//	    tp.CommitTransaction(ctx)
//	} else {
//	    tp.AbortTransaction(ctx)
//	}
type TransactionalProducer struct {
	client *kgo.Client
}

// NewTransactionalProducer creates a transactional producer with the given txID.
//
// txID must be unique per producer instance and stable across restarts.
// Convention: use "{service}-{partition}" e.g. "processor-0".
// If two instances share a txID the broker will fence the older one —
// this is intentional and prevents zombie producers after a failover.
func NewTransactionalProducer(cfg config.KafkaConfig, txID string) (*TransactionalProducer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.BootstrapServers),
		kgo.SASL(plain.Auth{
			User: cfg.APIKey,
			Pass: cfg.APISecret,
		}.AsMechanism()),
		kgo.DialTLS(),

		// A non-empty TransactionalID switches the producer into transactional
		// mode. The broker assigns a producer epoch and enforces fencing.
		kgo.TransactionalID(txID),

		// Transactional producers require acks=all — the broker rejects any
		// other ack level when a TransactionalID is set.
		kgo.RequiredAcks(kgo.AllISRAcks()),

		kgo.ProducerBatchCompression(kgo.Lz4Compression()),
		kgo.ProducerLinger(5*time.Millisecond),
		kgo.RecordPartitioner(kgo.StickyKeyPartitioner(nil)),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka transactional producer: creating client (txID=%q): %w", txID, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("kafka transactional producer: pinging broker (txID=%q): %w", txID, err)
	}

	return &TransactionalProducer{client: client}, nil
}

// BeginTransaction starts a new Kafka transaction.
// All records produced via Produce until CommitTransaction or AbortTransaction
// are part of this transaction and are invisible to consumers using
// read_committed isolation (the default).
func (tp *TransactionalProducer) BeginTransaction() error {
	if err := tp.client.BeginTransaction(); err != nil {
		return fmt.Errorf("kafka transactional producer: beginning transaction: %w", err)
	}
	return nil
}

// Produce adds a record to the current transaction.
// The record is buffered and sent asynchronously; call CommitTransaction to
// flush and commit, or AbortTransaction to discard.
func (tp *TransactionalProducer) Produce(ctx context.Context, topic string, key, value []byte) error {
	results := tp.client.ProduceSync(ctx, &kgo.Record{
		Topic: topic,
		Key:   key,
		Value: value,
	})
	if err := results.FirstErr(); err != nil {
		return fmt.Errorf("kafka transactional producer: producing to %q: %w", topic, err)
	}
	return nil
}

// CommitTransaction flushes all buffered records and commits the transaction.
// After this returns without error, the records are visible to downstream
// consumers using read_committed isolation.
func (tp *TransactionalProducer) CommitTransaction(ctx context.Context) error {
	if err := tp.client.EndTransaction(ctx, kgo.TryCommit); err != nil {
		return fmt.Errorf("kafka transactional producer: committing transaction: %w", err)
	}
	return nil
}

// AbortTransaction discards all buffered records for the current transaction.
// Call this on any processing error to avoid partial output landing in the
// output topic. The input offsets should NOT be committed — the message will
// be reprocessed on the next poll.
func (tp *TransactionalProducer) AbortTransaction(ctx context.Context) error {
	if err := tp.client.EndTransaction(ctx, kgo.TryAbort); err != nil {
		return fmt.Errorf("kafka transactional producer: aborting transaction: %w", err)
	}
	return nil
}

// Close flushes any pending records and closes the connection.
func (tp *TransactionalProducer) Close() {
	tp.client.Close()
}
