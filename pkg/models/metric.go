// Package models defines the core domain types shared across all TelemetryFlow
// services. These are plain Go structs with no framework dependencies.
//
// Types here map 1:1 to Avro schemas in pkg/schema/schemas/.
// They also map to the database tables defined in migrations/.
package models

import "time"

// DeviceMetric represents a single telemetry reading from an IoT device.
// This is the primary event type flowing through the pipeline.
//
// Data path: Simulator → device.metrics (Kafka) → Ingestor → TimescaleDB
//
// Field types use float32 to match the Avro schema (Avro "float" = 32-bit).
type DeviceMetric struct {
	DeviceID        string
	Timestamp       time.Time
	Temperature     float32
	Humidity        float32
	Pressure        float32
	BatteryLevel    float32
	FirmwareVersion string
	LocationID      string
}
