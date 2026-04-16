// Processor is the stream processing service. It consumes device.metrics,
// maintains per-device sliding windows, detects anomalies, enriches events
// with device registry data, and produces to device.aggregated and device.alerts.
//
// Key behaviours:
//   - Sliding window aggregations: tracks the last N seconds of readings
//     per device in memory. Windows are keyed by device_id, and because
//     Kafka partitions by device_id, all readings for a device always land
//     on the same processor instance — no cross-instance coordination needed.
//   - Anomaly detection: evaluates temperature high/low and battery low
//     thresholds on each window. On breach, emits a DeviceAlert to device.alerts.
//   - Enrichment: augments aggregates with device location_id from the
//     PostgreSQL device registry, cached locally with a TTL (ADR-006).
//   - Cooperative rebalancing: when a new processor instance joins the group,
//     only the reassigned partitions pause. In-memory window state for those
//     partitions is discarded; new state rebuilds from the next records.
//
// Scaling: run up to 12 instances (one per partition). Kafka distributes
// partitions using the cooperative sticky assignor (franz-go default).
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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// PostgreSQL pool — used by the enricher to look up device metadata.
	pgPool, err := db.NewPostgresPool(ctx, cfg.Postgres)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	// Schema Registry — shared between deserializer and publisher.
	registry, err := schema.NewRegistryClient(
		cfg.Kafka.SchemaRegistryURL,
		cfg.Kafka.SchemaRegistryAPIKey,
		cfg.Kafka.SchemaRegistryAPISecret,
	)
	if err != nil {
		slog.Error("failed to create schema registry client", "error", err)
		os.Exit(1)
	}

	// Kafka consumer reads from device.metrics.
	consumer, err := kafka.NewConsumer(cfg.Kafka, kafka.GroupProcessor, kafka.TopicDeviceMetrics)
	if err != nil {
		slog.Error("failed to create kafka consumer", "error", err)
		os.Exit(1)
	}

	// DLQ publisher for unprocessable records.
	dlqPublisher, err := kafka.NewDLQPublisher(cfg.Kafka, kafka.TopicMetricsDLQ)
	if err != nil {
		slog.Error("failed to create dlq publisher", "error", err)
		os.Exit(1)
	}

	// Publisher writes to device.aggregated and device.alerts.
	pub, err := newPublisher(cfg.Kafka, registry)
	if err != nil {
		slog.Error("failed to create publisher", "error", err)
		os.Exit(1)
	}

	windowSize := time.Duration(cfg.Processor.WindowSizeSeconds) * time.Second
	cacheTTL := time.Duration(cfg.Processor.EnrichmentCacheTTLSecs) * time.Second

	pc := newProcessorConsumer(
		consumer,
		dlqPublisher,
		schema.NewDeserializer(registry),
		newWindowStore(windowSize),
		newAnomalyDetector(
			cfg.Processor.AnomalyTempHigh,
			cfg.Processor.AnomalyTempLow,
			cfg.Processor.AnomalyBatteryLow,
		),
		newEnricher(pgPool, cacheTTL),
		pub,
	)

	slog.Info("processor starting",
		"consumer_group", kafka.GroupProcessor,
		"topic", kafka.TopicDeviceMetrics,
		"window_seconds", cfg.Processor.WindowSizeSeconds,
		"temp_high_threshold", cfg.Processor.AnomalyTempHigh,
		"temp_low_threshold", cfg.Processor.AnomalyTempLow,
		"battery_low_threshold", cfg.Processor.AnomalyBatteryLow,
	)

	if err := pc.Run(ctx); err != nil {
		slog.Error("processor run error", "error", err)
	}

	slog.Info("shutdown signal received, closing connections...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := pc.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("processor stopped cleanly")
}
