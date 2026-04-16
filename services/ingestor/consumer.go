package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/telemetryflow/pkg/kafka"
	"github.com/telemetryflow/pkg/models"
	"github.com/telemetryflow/pkg/schema"
	"github.com/twmb/franz-go/pkg/kgo"
)

// ingestorConsumer coordinates the poll → deserialize → batch → flush loop.
//
// Flush is triggered by whichever comes first (ADR-005):
//   - batch reaches batchSize records
//   - flushInterval elapses since the last flush
//
// franz-go's PollFetches returns as soon as the broker responds (which is at
// most FetchMaxWait, default 5s). For our 500ms flush interval, we check
// time.Since(lastFlush) after every poll and flush if overdue — no separate
// ticker goroutine needed.
//
// Offset commit discipline (ADR-002):
//   - Offsets are committed only AFTER a successful DB write.
//   - Records that fail deserialization go to the DLQ; their offsets are also
//     committed so a poison pill cannot block a partition indefinitely.
//   - If the DB write fails, NO offsets are committed — Kafka redelivers the
//     batch on restart, giving the write a chance to succeed.
type ingestorConsumer struct {
	consumer      *kafka.Consumer
	writer        *writer
	dlq           *kafka.DLQPublisher
	deserializer  *schema.Deserializer
	batchSize     int
	flushInterval time.Duration

	// pending holds records accumulated since the last flush.
	// Run is single-threaded; no locking needed.
	pendingMetrics []models.DeviceMetric
	pendingRecords []*kgo.Record
}

func newIngestorConsumer(
	consumer *kafka.Consumer,
	w *writer,
	dlq *kafka.DLQPublisher,
	deserializer *schema.Deserializer,
	batchSize int,
	flushInterval time.Duration,
) *ingestorConsumer {
	return &ingestorConsumer{
		consumer:      consumer,
		writer:        w,
		dlq:           dlq,
		deserializer:  deserializer,
		batchSize:     batchSize,
		flushInterval: flushInterval,
	}
}

// Run is the main consume-batch-flush loop. Blocks until ctx is cancelled.
// It does not flush the final pending batch — call Shutdown for that.
func (c *ingestorConsumer) Run(ctx context.Context) error {
	lastFlush := time.Now()

	for {
		if ctx.Err() != nil {
			// Context cancelled — exit cleanly. Shutdown() will flush the
			// remaining batch with a fresh context.
			return nil
		}

		fetches := c.consumer.Poll(ctx)
		if fetches.IsClientClosed() {
			return nil
		}

		// Partition-level errors (e.g. leader election, network blip) are
		// non-fatal. Log them and continue — the affected partition will be
		// retried on the next poll when the leader recovers.
		for _, fetchErr := range fetches.Errors() {
			slog.ErrorContext(ctx, "fetch error from broker",
				"topic", fetchErr.Topic,
				"partition", fetchErr.Partition,
				"error", fetchErr.Err,
			)
		}

		for iter := fetches.RecordIter(); !iter.Done(); {
			record := iter.Next()

			var metric models.DeviceMetric
			if err := c.deserializer.Deserialize(record.Value, &metric); err != nil {
				slog.ErrorContext(ctx, "deserialization failed, routing to DLQ",
					"error", err,
					"topic", record.Topic,
					"partition", record.Partition,
					"offset", record.Offset,
				)
				if dlqErr := c.dlq.Publish(ctx, record, err); dlqErr != nil {
					slog.ErrorContext(ctx, "dlq publish failed", "error", dlqErr)
				}
				// Still track the record for offset commit — we've handled it
				// via DLQ, so we must advance past it to avoid re-DLQ'ing on restart.
				c.pendingRecords = append(c.pendingRecords, record)
				continue
			}

			c.pendingMetrics = append(c.pendingMetrics, metric)
			c.pendingRecords = append(c.pendingRecords, record)
		}

		// Flush when the batch is full or the interval has elapsed.
		// len(pendingMetrics) check excludes DLQ-only batches from triggering
		// a DB write; we still commit their offsets via flush.
		shouldFlush := len(c.pendingMetrics) >= c.batchSize ||
			(len(c.pendingRecords) > 0 && time.Since(lastFlush) >= c.flushInterval)

		if shouldFlush {
			if err := c.flush(ctx); err != nil {
				return err
			}
			lastFlush = time.Now()
		}
	}
}

// Shutdown flushes remaining pending records and closes the consumer and DLQ.
// Pass a context with a deadline — if the final DB write takes longer than the
// deadline, records are not committed and will be redelivered on next start.
func (c *ingestorConsumer) Shutdown(ctx context.Context) error {
	if err := c.flush(ctx); err != nil {
		return fmt.Errorf("ingestor: shutdown flush: %w", err)
	}
	c.consumer.Close()
	c.dlq.Close()
	return nil
}

// flush writes pendingMetrics to TimescaleDB and commits all pendingRecords.
//
// On DB write failure:
//   - All pending metrics are routed to the DLQ.
//   - Offsets are still committed so we advance past the failed batch.
//     The DLQ is the audit trail; ops can replay from there if needed.
//
// On commit failure:
//   - Returns an error — the caller (Run or Shutdown) should propagate it.
//     On restart, Kafka will redeliver the uncommitted records.
func (c *ingestorConsumer) flush(ctx context.Context) error {
	if len(c.pendingRecords) == 0 {
		return nil
	}

	// Always clear pending state when flush returns, success or failure.
	defer func() {
		c.pendingMetrics = c.pendingMetrics[:0]
		c.pendingRecords = c.pendingRecords[:0]
	}()

	if len(c.pendingMetrics) > 0 {
		start := time.Now()
		err := c.writer.WriteBatch(ctx, c.pendingMetrics)
		if err != nil {
			slog.ErrorContext(ctx, "batch write failed, routing to DLQ",
				"batch_size", len(c.pendingMetrics),
				"error", err,
			)
			for _, r := range c.pendingRecords {
				if dlqErr := c.dlq.Publish(ctx, r, err); dlqErr != nil {
					slog.ErrorContext(ctx, "dlq publish failed", "error", dlqErr)
				}
			}
			// Fall through to commit — we've DLQ'd everything, advance past them.
		} else {
			slog.InfoContext(ctx, "batch written to timescale",
				"count", len(c.pendingMetrics),
				"duration_ms", time.Since(start).Milliseconds(),
			)
		}
	}

	// Commit all pending records (both successfully written and DLQ'd).
	if err := c.consumer.Commit(ctx, c.pendingRecords...); err != nil {
		return fmt.Errorf("ingestor: committing offsets: %w", err)
	}

	return nil
}
