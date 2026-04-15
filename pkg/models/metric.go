package models

import "time"

// DeviceMetric represents a single telemetry reading from an IoT device.
// This is the primary event type flowing through the pipeline.
//
// Data path: Simulator → device.metrics (Kafka) → Ingestor → TimescaleDB
//
// Avro field names (snake_case) are mapped via struct tags.
// hamba/avro uses these tags to encode/decode the binary payload.
// time.Time maps to Avro long + logicalType:timestamp-millis automatically.
type DeviceMetric struct {
	DeviceID        string    `avro:"device_id"`
	Timestamp       time.Time `avro:"timestamp"`
	Temperature     float32   `avro:"temperature"`
	Humidity        float32   `avro:"humidity"`
	Pressure        float32   `avro:"pressure"`
	BatteryLevel    float32   `avro:"battery_level"`
	FirmwareVersion string    `avro:"firmware_version"`
	LocationID      string    `avro:"location_id"`
}
