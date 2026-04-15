package models

import "time"

// AlertType classifies what kind of anomaly triggered the alert.
// These map directly to the AlertType enum in device_alert.avsc.
//
// IMPORTANT: This type is part of the public contract on device.alerts.
// Never remove or change existing values — only add new ones (ADR-007 + CLAUDE.md).
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
// Field types use float32/int32 to match the Avro schema.
type DeviceAlert struct {
	AlertID        string
	DeviceID       string
	AlertType      AlertType
	Severity       Severity
	TriggeredAt    time.Time
	MetricValue    float32
	ThresholdValue float32
	WindowSeconds  int32
}
