package models

import "time"

// AlertType classifies what kind of anomaly triggered the alert.
// These map directly to the AlertType enum in device_alert.avsc.
//
// IMPORTANT: This type is part of the public contract on device.alerts.
// Never remove or change existing values — only add new ones (CLAUDE.md).
type AlertType string

const (
	AlertTypeTempHigh   AlertType = "TEMP_HIGH"
	AlertTypeTempLow    AlertType = "TEMP_LOW"
	AlertTypeBatteryLow AlertType = "BATTERY_LOW"
	AlertTypeOffline    AlertType = "OFFLINE"
	AlertTypeSpike      AlertType = "SPIKE"
)

// Severity indicates how urgently the alert should be acted on.
// These map directly to the Severity enum in device_alert.avsc.
//
// IMPORTANT: Same contract rule — never remove or rename, only add.
type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

// DeviceAlert represents an anomaly detected by the stream processor.
//
// Data path: Processor → device.alerts (Kafka) → Sink → PostgreSQL alerts table
//                                               → External notification service
//
// AlertType and Severity are Avro enum types — hamba/avro serializes them
// as strings and validates against the enum symbols in the schema.
type DeviceAlert struct {
	AlertID        string    `avro:"alert_id"`
	DeviceID       string    `avro:"device_id"`
	AlertType      AlertType `avro:"alert_type"`
	Severity       Severity  `avro:"severity"`
	TriggeredAt    time.Time `avro:"triggered_at"`
	MetricValue    float32   `avro:"metric_value"`
	ThresholdValue float32   `avro:"threshold_value"`
	WindowSeconds  int32     `avro:"window_seconds"`
}
