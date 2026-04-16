package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/telemetryflow/config"
	"github.com/telemetryflow/pkg/db"
	"github.com/telemetryflow/pkg/kafka"
	"github.com/telemetryflow/pkg/logger"
	"github.com/telemetryflow/pkg/models"
	"github.com/telemetryflow/pkg/schema"
)

func main() {
	// Set up structured JSON logger first — all subsequent errors are logged, not printed.
	log := logger.New()
	slog.SetDefault(log)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Graceful shutdown: SIGINT (Ctrl+C) or SIGTERM cancels ctx.
	// All device goroutines watch ctx.Done() and stop cleanly.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	producer, err := kafka.NewProducer(cfg.Kafka)
	if err != nil {
		slog.Error("failed to create kafka producer", "error", err)
		os.Exit(1)
	}
	defer producer.Close()

	registry, err := schema.NewRegistryClient(
		cfg.Kafka.SchemaRegistryURL,
		cfg.Kafka.SchemaRegistryAPIKey,
		cfg.Kafka.SchemaRegistryAPISecret,
	)
	if err != nil {
		slog.Error("failed to create schema registry client", "error", err)
		os.Exit(1)
	}

	// Register the DeviceMetric schema on startup. If the schema already
	// exists on the registry the existing ID is returned — idempotent.
	serializer, err := schema.NewSerializer(registry, "device.metrics-value", schema.DeviceMetricSchema)
	if err != nil {
		slog.Error("failed to create avro serializer", "error", err)
		os.Exit(1)
	}

	// Connect to PostgreSQL to load the device registry. The registry is the
	// source of truth for which devices exist — we never generate IDs here.
	pgPool, err := db.NewPostgresPool(ctx, cfg.Postgres)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	registeredDevices, err := db.LoadActiveDevices(ctx, pgPool)
	if err != nil {
		slog.Error("failed to load active devices from registry", "error", err)
		os.Exit(1)
	}
	if len(registeredDevices) == 0 {
		slog.Error("no active devices found in registry — run 'psql telemetryflow < scripts/seed-devices.sql' first")
		os.Exit(1)
	}

	gen := newGenerator(cfg.Simulator.ChaosMode)
	interval := time.Duration(cfg.Simulator.EmitIntervalMS) * time.Millisecond

	slog.Info("simulator starting",
		"device_count", len(registeredDevices),
		"emit_interval_ms", cfg.Simulator.EmitIntervalMS,
		"chaos_mode", cfg.Simulator.ChaosMode,
	)

	// Spawn one goroutine per device from the registry. Each goroutine owns
	// its device state exclusively — no shared mutable state, no locks needed.
	var wg sync.WaitGroup
	for _, reg := range registeredDevices {
		wg.Add(1)

		d := newDevice(reg.DeviceID, reg.LocationID, reg.FirmwareVersion)

		go func(dev *device) {
			defer wg.Done()
			runDevice(ctx, dev, gen, producer, serializer, interval)
		}(d)
	}

	// Block until shutdown signal.
	<-ctx.Done()
	slog.Info("shutdown signal received, waiting for device goroutines to stop...")

	wg.Wait()
	slog.Info("simulator stopped cleanly")
}

// runDevice is the per-device goroutine. It ticks on an interval, generates
// a metric, and produces it to Kafka. Stops when ctx is cancelled.
//
// The ticker — not time.Sleep — is used so shutdown is responsive.
// time.Sleep would block the goroutine for the full interval after cancel.
// A ticker + select returns immediately when ctx.Done() fires.
//
// A small random jitter is added to the first tick so 1000 goroutines
// don't all wake up and hammer Kafka at exactly the same millisecond.
func runDevice(
	ctx context.Context,
	d *device,
	gen *generator,
	producer *kafka.Producer,
	serializer *schema.Serializer,
	interval time.Duration,
) {
	// Jitter: spread the initial tick across one full interval.
	// Without this, all devices emit simultaneously causing a burst.
	jitter := time.Duration(float64(interval) * randFloat())
	select {
	case <-ctx.Done():
		return
	case <-time.After(jitter):
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metric := gen.generate(d)
			if err := emit(ctx, producer, serializer, metric); err != nil {
				slog.ErrorContext(ctx, "failed to emit metric",
					"device_id", d.id,
					"error", err,
				)
				// Do not return — a transient Kafka error should not kill the device goroutine.
				// franz-go retries internally; log and continue to the next tick.
			}
		}
	}
}

// emit serializes a DeviceMetric to Avro wire format and produces it to Kafka.
// The device_id is used as the partition key so all metrics for a device
// always land on the same partition (preserving per-device ordering).
func emit(
	ctx context.Context,
	producer *kafka.Producer,
	serializer *schema.Serializer,
	metric models.DeviceMetric,
) error {
	value, err := serializer.Serialize(metric)
	if err != nil {
		return fmt.Errorf("simulator: serializing metric for %s: %w", metric.DeviceID, err)
	}

	if err := producer.Produce(ctx, kafka.TopicDeviceMetrics, []byte(metric.DeviceID), value); err != nil {
		return fmt.Errorf("simulator: producing to kafka for %s: %w", metric.DeviceID, err)
	}

	return nil
}

// randFloat returns a random float in [0, 1) using the global rand source.
// Extracted to make the jitter calculation readable.
func randFloat() float64 {
	return rand.Float64() //nolint:gosec // non-cryptographic use
}
