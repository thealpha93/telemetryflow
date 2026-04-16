package main

import (
	"context"
	"fmt"

	"github.com/telemetryflow/config"
	"github.com/telemetryflow/pkg/kafka"
	"github.com/telemetryflow/pkg/models"
	"github.com/telemetryflow/pkg/schema"
)

// publisher encodes and publishes to the processor's two output topics.
//
// A single idempotent Producer is shared for both topics — franz-go routes
// records to the correct topic based on the Record.Topic field, so there's
// no need for separate producer connections (ADR-003).
//
// Per-device ordering is preserved: both aggregates and alerts are keyed by
// device_id, so they land on the same partition in their respective topics.
type publisher struct {
	producer        *kafka.Producer
	aggSerializer   *schema.Serializer
	alertSerializer *schema.Serializer
}

// newPublisher creates a publisher that registers both schemas and connects
// to the broker. Fails fast if the broker or schema registry is unreachable.
func newPublisher(cfg config.KafkaConfig, registry *schema.RegistryClient) (*publisher, error) {
	producer, err := kafka.NewProducer(cfg)
	if err != nil {
		return nil, fmt.Errorf("publisher: creating producer: %w", err)
	}

	aggSer, err := schema.NewSerializer(registry, "device.aggregated-value", schema.DeviceAggregateSchema)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("publisher: registering aggregate schema: %w", err)
	}

	alertSer, err := schema.NewSerializer(registry, "device.alerts-value", schema.DeviceAlertSchema)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("publisher: registering alert schema: %w", err)
	}

	return &publisher{
		producer:        producer,
		aggSerializer:   aggSer,
		alertSerializer: alertSer,
	}, nil
}

// PublishAggregate encodes agg as Confluent Avro and produces it to
// device.aggregated. Keyed by device_id for per-device partition ordering.
func (p *publisher) PublishAggregate(ctx context.Context, agg models.DeviceAggregate) error {
	value, err := p.aggSerializer.Serialize(agg)
	if err != nil {
		return fmt.Errorf("publisher: serializing aggregate for %s: %w", agg.DeviceID, err)
	}

	if err := p.producer.Produce(ctx, kafka.TopicDeviceAggregated, []byte(agg.DeviceID), value); err != nil {
		return fmt.Errorf("publisher: producing aggregate for %s: %w", agg.DeviceID, err)
	}

	return nil
}

// PublishAlert encodes alert as Confluent Avro and produces it to
// device.alerts. This topic is a public contract — schema is immutable.
func (p *publisher) PublishAlert(ctx context.Context, alert models.DeviceAlert) error {
	value, err := p.alertSerializer.Serialize(alert)
	if err != nil {
		return fmt.Errorf("publisher: serializing alert %s: %w", alert.AlertID, err)
	}

	if err := p.producer.Produce(ctx, kafka.TopicDeviceAlerts, []byte(alert.DeviceID), value); err != nil {
		return fmt.Errorf("publisher: producing alert %s: %w", alert.AlertID, err)
	}

	return nil
}

// Close flushes buffered records and closes the producer connection.
func (p *publisher) Close() {
	p.producer.Close()
}
