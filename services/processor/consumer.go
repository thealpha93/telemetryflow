package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/telemetryflow/pkg/kafka"
	"github.com/telemetryflow/pkg/models"
	"github.com/telemetryflow/pkg/schema"
	"github.com/twmb/franz-go/pkg/kgo"
)

// processorConsumer drives the poll → process → publish loop.
//
// Per-record processing pipeline (in order):
//  1. Deserialize Avro → DeviceMetric
//  2. Push into the device's sliding window → windowAggregate
//  3. Enrich aggregate with device metadata (location_id) from cache
//  4. Publish aggregate to device.aggregated
//  5. Run anomaly detection on the aggregate
//  6. Publish any new alerts to device.alerts
//  7. Commit the input offset — ONLY after all publishes succeed
//
// On failure: route to device.metrics.dlq and commit the offset to
// advance past the poison pill. The DLQ is the audit trail for replay.
//
// Commit discipline: offsets are committed in a batch at the end of each
// poll, but only if ctx is still valid — same pattern as the ingestor fix.
type processorConsumer struct {
	consumer     *kafka.Consumer
	dlq          *kafka.DLQPublisher
	deserializer *schema.Deserializer
	windows      *windowStore
	detector     *anomalyDetector
	enricher     *enricher
	publisher    *publisher
}

// newProcessorConsumer constructs the consumer with all its dependencies
// already initialised. The caller is responsible for creating each component.
func newProcessorConsumer(
	consumer *kafka.Consumer,
	dlq *kafka.DLQPublisher,
	deserializer *schema.Deserializer,
	windows *windowStore,
	detector *anomalyDetector,
	enricher *enricher,
	publisher *publisher,
) *processorConsumer {
	return &processorConsumer{
		consumer:     consumer,
		dlq:          dlq,
		deserializer: deserializer,
		windows:      windows,
		detector:     detector,
		enricher:     enricher,
		publisher:    publisher,
	}
}

// Run is the main poll loop. Blocks until ctx is cancelled.
// No pending batch to flush on exit — each poll's records are fully
// processed and committed before the next poll begins.
func (c *processorConsumer) Run(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		fetches := c.consumer.Poll(ctx)
		if fetches.IsClientClosed() {
			return nil
		}

		for _, fetchErr := range fetches.Errors() {
			slog.ErrorContext(ctx, "fetch error from broker",
				"topic", fetchErr.Topic,
				"partition", fetchErr.Partition,
				"error", fetchErr.Err,
			)
		}

		var toCommit []*kgo.Record
		for iter := fetches.RecordIter(); !iter.Done(); {
			record := iter.Next()

			if err := c.processRecord(ctx, record); err != nil {
				slog.ErrorContext(ctx, "record processing failed, routing to DLQ",
					"error", err,
					"topic", record.Topic,
					"partition", record.Partition,
					"offset", record.Offset,
				)
				if dlqErr := c.dlq.Publish(ctx, record, err); dlqErr != nil {
					slog.ErrorContext(ctx, "dlq publish failed", "error", dlqErr)
				}
			}
			// Always include in toCommit — DLQ'd records must advance past too.
			toCommit = append(toCommit, record)
		}

		// Same pattern as the ingestor fix: don't commit with a cancelled ctx.
		// Uncommitted records will be redelivered on restart — correct behaviour.
		if len(toCommit) > 0 && ctx.Err() == nil {
			if err := c.consumer.Commit(ctx, toCommit...); err != nil {
				slog.ErrorContext(ctx, "failed to commit offsets", "error", err)
				// Don't return — a commit failure is retried on the next poll.
			}
		}
	}
}

// Shutdown closes the consumer, publisher, and DLQ producer.
// No pending state needs draining (the processor has no batch accumulator).
func (c *processorConsumer) Shutdown(ctx context.Context) error {
	c.publisher.Close()
	c.dlq.Close()
	c.consumer.Close()
	return nil
}

// processRecord runs the full per-record pipeline for a single Kafka record.
// Returns an error if any step fails; the caller routes failed records to DLQ.
func (c *processorConsumer) processRecord(ctx context.Context, record *kgo.Record) error {
	// Step 1: Deserialize.
	var metric models.DeviceMetric
	if err := c.deserializer.Deserialize(record.Value, &metric); err != nil {
		return fmt.Errorf("processor: deserializing metric: %w", err)
	}

	// Step 2: Update sliding window.
	agg, err := c.windows.push(metric)
	if err != nil {
		return fmt.Errorf("processor: updating window for %s: %w", metric.DeviceID, err)
	}

	// Step 3: Enrich with device registry metadata.
	// Best-effort: if the enricher fails (cold cache + DB down), log and
	// continue with an empty location_id rather than dropping the record.
	device, enrichErr := c.enricher.GetDevice(ctx, metric.DeviceID)
	if enrichErr != nil {
		slog.WarnContext(ctx, "enrichment failed, proceeding without device metadata",
			"device_id", metric.DeviceID,
			"error", enrichErr,
		)
	}

	// Step 4: Publish aggregate to device.aggregated.
	domainAgg := models.DeviceAggregate{
		DeviceID:    agg.deviceID,
		LocationID:  device.LocationID, // empty string if enrichment failed
		WindowStart: agg.windowStart,
		WindowEnd:   agg.windowEnd,
		AvgTemp:     agg.avgTemp,
		MaxTemp:     agg.maxTemp,
		MinTemp:     agg.minTemp,
		AvgBattery:  agg.avgBattery,
		SampleCount: int32(agg.sampleCount),
	}
	if err := c.publisher.PublishAggregate(ctx, domainAgg); err != nil {
		return fmt.Errorf("processor: publishing aggregate for %s: %w", metric.DeviceID, err)
	}

	// Step 5+6: Detect anomalies and publish alerts.
	for _, alert := range c.detector.detect(agg) {
		if err := c.publisher.PublishAlert(ctx, alert); err != nil {
			return fmt.Errorf("processor: publishing alert %s for %s: %w",
				alert.AlertType, metric.DeviceID, err)
		}
		slog.InfoContext(ctx, "alert emitted",
			"device_id", alert.DeviceID,
			"alert_type", alert.AlertType,
			"severity", alert.Severity,
			"metric_value", alert.MetricValue,
			"threshold", alert.ThresholdValue,
		)
	}

	return nil
}
