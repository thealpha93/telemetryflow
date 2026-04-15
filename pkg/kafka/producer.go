package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/telemetryflow/config"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
)

// Producer is an idempotent franz-go producer wrapper.
//
// Every option here maps to an explicit architectural decision:
//
//	SeedBrokers      — the Confluent Cloud bootstrap server list
//	SASL + DialTLS   — PLAIN auth over TLS (required by Confluent Cloud)
//	AllISRAcks       — acks=all: leader + all in-sync replicas must confirm
//	                   before Produce returns. Prevents data loss on leader failure.
//	Lz4Compression   — compresses batches before sending. Reduces network bytes
//	                   to Confluent Cloud and lowers egress cost.
//	ProducerLinger   — wait up to 5ms to accumulate records into a batch.
//	                   Trades a tiny bit of latency for much better throughput.
//	StickyKeyPartitioner — all records with the same key go to the same partition.
//	                        device_id is the key, so all metrics for a device
//	                        are ordered within a single partition.
//
// Idempotency (ADR-003) is enabled by default in franz-go when AllISRAcks
// is set — the broker assigns a producer ID and sequence numbers, so retried
// records are deduplicated server-side even if the network dropped the ack.
type Producer struct {
	client *kgo.Client
}

// NewProducer creates a connected idempotent Producer.
// It pings the broker on startup to catch misconfigured credentials early —
// failing fast here is better than failing silently on first produce.
func NewProducer(cfg config.KafkaConfig) (*Producer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.BootstrapServers),
		kgo.SASL(plain.Auth{
			User: cfg.APIKey,
			Pass: cfg.APISecret,
		}.AsMechanism()),
		kgo.DialTLS(),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.ProducerBatchCompression(kgo.Lz4Compression()),
		kgo.ProducerLinger(5*time.Millisecond),
		kgo.RecordPartitioner(kgo.StickyKeyPartitioner(nil)),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka producer: creating client: %w", err)
	}

	// Verify the broker is reachable and credentials are valid.
	// Without this, a misconfigured bootstrap server or wrong API key
	// would only surface on the first Produce call, deep in the pipeline.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("kafka producer: pinging broker at %q: %w", cfg.BootstrapServers, err)
	}

	return &Producer{client: client}, nil
}

// Produce sends a single record to the given topic synchronously.
// It blocks until the record is acknowledged by all in-sync replicas.
//
// key is the partition key — always pass the device_id as bytes so that
// all records for a device land on the same partition (preserving order).
//
// value must be Confluent Avro wire-format bytes from schema.Serializer.
func (p *Producer) Produce(ctx context.Context, topic string, key, value []byte) error {
	results := p.client.ProduceSync(ctx, &kgo.Record{
		Topic: topic,
		Key:   key,
		Value: value,
	})
	if err := results.FirstErr(); err != nil {
		return fmt.Errorf("kafka producer: producing to topic %q: %w", topic, err)
	}
	return nil
}

// Close flushes any buffered records and closes the connection.
// Always call in a defer or shutdown handler — unflushed records are lost.
func (p *Producer) Close() {
	p.client.Close()
}
