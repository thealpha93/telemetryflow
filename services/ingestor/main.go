// Ingestor consumes raw device metrics from device.metrics and writes them
// to the TimescaleDB device_metrics hypertable using batched inserts (ADR-005).
//
// Key behaviours:
//   - At-least-once delivery with idempotent writes: offsets are only committed
//     after a successful DB write. A crash mid-batch causes reprocessing on
//     restart, which is safe — the hypertable has no PRIMARY KEY constraint
//     and duplicates are prevented by the idempotent producer upstream.
//   - Batched TimescaleDB writes via COPY protocol: 500 records per flush or
//     500ms, whichever comes first (ADR-005).
//   - DLQ: records that fail deserialization are published to device.metrics.dlq
//     with error headers (ADR-007). Records that fail DB writes are also DLQ'd
//     after the write fails, and their offsets are committed.
//   - Graceful shutdown: on SIGTERM/SIGINT, Run() exits, Shutdown() flushes
//     the remaining batch and closes all connections before the process exits.
//
// Scaling: run multiple instances with the same consumer group (GroupIngestor).
// Kafka distributes the 12 device.metrics partitions across instances.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/telemetryflow/config"
	"github.com/telemetryflow/pkg/db"
	"github.com/telemetryflow/pkg/kafka"
	"github.com/telemetryflow/pkg/logger"
	"github.com/telemetryflow/pkg/schema"
)

func main() {
	log := logger.New()
	slog.SetDefault(log)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Graceful shutdown: SIGINT/SIGTERM cancels ctx, signalling Run to exit.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// TimescaleDB — destination for all raw device metrics.
	tsPool, err := db.NewTimescalePool(ctx, cfg.Timescale)
	if err != nil {
		slog.Error("failed to connect to timescale", "error", err)
		os.Exit(1)
	}
	defer tsPool.Close()

	w := newWriter(tsPool)

	// Kafka consumer: reads from device.metrics, manual commits (ADR-002).
	consumer, err := kafka.NewConsumer(cfg.Kafka, kafka.GroupIngestor, kafka.TopicDeviceMetrics)
	if err != nil {
		slog.Error("failed to create kafka consumer", "error", err)
		os.Exit(1)
	}

	// DLQ publisher: routes unprocessable records to device.metrics.dlq (ADR-007).
	dlqPublisher, err := kafka.NewDLQPublisher(cfg.Kafka, kafka.TopicMetricsDLQ)
	if err != nil {
		slog.Error("failed to create dlq publisher", "error", err)
		os.Exit(1)
	}

	// Schema Registry + Avro deserializer.
	// The deserializer fetches schemas by ID from the registry on first use
	// and caches them — subsequent decodes make no network calls.
	registry, err := schema.NewRegistryClient(
		cfg.Kafka.SchemaRegistryURL,
		cfg.Kafka.SchemaRegistryAPIKey,
		cfg.Kafka.SchemaRegistryAPISecret,
	)
	if err != nil {
		slog.Error("failed to create schema registry client", "error", err)
		os.Exit(1)
	}
	deserializer := schema.NewDeserializer(registry)

	ic := newIngestorConsumer(
		consumer,
		w,
		dlqPublisher,
		deserializer,
		cfg.Ingestor.BatchSize,
		time.Duration(cfg.Ingestor.FlushIntervalMS)*time.Millisecond,
	)

	slog.Info("ingestor starting",
		"consumer_group", kafka.GroupIngestor,
		"topic", kafka.TopicDeviceMetrics,
		"batch_size", cfg.Ingestor.BatchSize,
		"flush_interval_ms", cfg.Ingestor.FlushIntervalMS,
	)

	if err := ic.Run(ctx); err != nil {
		slog.Error("ingestor run error", "error", err)
		// Do not exit yet — attempt a clean shutdown to flush the pending batch.
	}

	slog.Info("shutdown signal received, draining pending batch...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := ic.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("ingestor stopped cleanly")
}
