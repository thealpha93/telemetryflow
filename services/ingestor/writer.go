package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/telemetryflow/pkg/models"
)

// writer handles batched inserts of DeviceMetric records into the
// TimescaleDB device_metrics hypertable using the PostgreSQL COPY protocol.
//
// Why COPY instead of INSERT?
// At 1000 devices/sec with batch size 500, the ingestor runs ~2 flushes/sec.
// COPY bypasses SQL parsing, the planner, and individual WAL records — it
// streams rows directly into the table in one round-trip. In practice COPY
// is 5-10x faster than batched INSERTs for write-heavy workloads like this.
//
// pgx's CopyFrom implements the binary COPY protocol automatically;
// we just provide column names and row data.
type writer struct {
	pool *pgxpool.Pool
}

// newWriter creates a writer that uses pool for all TimescaleDB writes.
func newWriter(pool *pgxpool.Pool) *writer {
	return &writer{pool: pool}
}

// columns must match the device_metrics table definition in
// migrations/timescale/001_create_device_metrics.sql exactly.
var deviceMetricsColumns = []string{
	"time", "device_id",
	"temperature", "humidity", "pressure", "battery_level",
	"firmware_version", "location_id",
}

// WriteBatch inserts metrics into device_metrics in a single COPY call.
// Returns immediately if metrics is empty.
func (w *writer) WriteBatch(ctx context.Context, metrics []models.DeviceMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	// Build the row source that pgx will stream to the server.
	// Each row must have values in the same order as deviceMetricsColumns.
	rows := make([][]any, len(metrics))
	for i, m := range metrics {
		rows[i] = []any{
			m.Timestamp, m.DeviceID,
			m.Temperature, m.Humidity, m.Pressure, m.BatteryLevel,
			m.FirmwareVersion, m.LocationID,
		}
	}

	n, err := w.pool.CopyFrom(
		ctx,
		pgx.Identifier{"device_metrics"},
		deviceMetricsColumns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("writer: copy into device_metrics (%d rows): %w", len(metrics), err)
	}
	if int(n) != len(metrics) {
		// This should never happen with CopyFromRows unless the server rejects
		// specific rows — surface it loudly so it isn't silently lost.
		return fmt.Errorf("writer: expected %d rows copied, got %d", len(metrics), n)
	}

	return nil
}
