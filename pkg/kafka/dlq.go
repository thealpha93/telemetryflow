package kafka

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/telemetryflow/config"
	"github.com/twmb/franz-go/pkg/kgo"
)

// DLQPublisher routes unprocessable messages to a dead-letter topic (ADR-007).
//
// A DLQ record is the original message payload plus five Kafka headers that
// capture why and where it failed:
//
//	error_reason        — human-readable description (the error string)
//	failed_at           — RFC3339 UTC timestamp of the failure
//	original_topic      — topic the message was consumed from
//	original_partition  — partition number as a decimal string
//	original_offset     — message offset as a decimal string
//
// This is enough information to inspect the record and replay it later
// without redeploying the service.
//
// DLQ publish errors are returned to the caller but must never block the
// main pipeline — log the error, increment a counter, and continue.
type DLQPublisher struct {
	producer *Producer
	topic    string
}

// NewDLQPublisher creates a DLQPublisher that routes failed records to dlqTopic.
// It reuses the standard idempotent Producer — DLQ writes are not transactional
// because the risk of a duplicate DLQ entry is far lower than blocking the pipeline.
func NewDLQPublisher(cfg config.KafkaConfig, dlqTopic string) (*DLQPublisher, error) {
	producer, err := NewProducer(cfg)
	if err != nil {
		return nil, fmt.Errorf("kafka dlq: creating producer for topic %q: %w", dlqTopic, err)
	}
	return &DLQPublisher{
		producer: producer,
		topic:    dlqTopic,
	}, nil
}

// Publish sends original to the DLQ topic with error metadata headers.
//
// The original record's key and value are preserved unchanged so that replays
// can re-route the message back to the primary topic and retry processing.
func (d *DLQPublisher) Publish(ctx context.Context, original *kgo.Record, reason error) error {
	headers := []kgo.RecordHeader{
		{Key: "error_reason", Value: []byte(reason.Error())},
		{Key: "failed_at", Value: []byte(time.Now().UTC().Format(time.RFC3339))},
		{Key: "original_topic", Value: []byte(original.Topic)},
		{Key: "original_partition", Value: []byte(strconv.Itoa(int(original.Partition)))},
		{Key: "original_offset", Value: []byte(strconv.FormatInt(original.Offset, 10))},
	}

	// Build a new record targeting the DLQ topic. The original headers are
	// appended after the error metadata so nothing from the source is lost.
	dlqRecord := &kgo.Record{
		Topic:   d.topic,
		Key:     original.Key,
		Value:   original.Value,
		Headers: append(headers, original.Headers...),
	}

	results := d.producer.client.ProduceSync(ctx, dlqRecord)
	if err := results.FirstErr(); err != nil {
		return fmt.Errorf("kafka dlq: publishing to %q: %w", d.topic, err)
	}
	return nil
}

// Close flushes any buffered DLQ records and closes the underlying producer.
func (d *DLQPublisher) Close() {
	d.producer.Close()
}
