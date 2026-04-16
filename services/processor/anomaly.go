package main

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/telemetryflow/pkg/models"
)

// anomalyDetector evaluates rules against a window aggregate and returns
// alerts to emit to device.alerts.
//
// Rules (thresholds configured via env vars):
//
//	TEMP_HIGH    avg_temp > tempHigh      → severity HIGH
//	TEMP_LOW     avg_temp < tempLow       → severity MEDIUM
//	BATTERY_LOW  avg_battery < batteryLow → severity HIGH
//	SPIKE        max_temp - min_temp > 20 → severity CRITICAL  (high intra-window variance)
//
// Deduplication: only emit an alert when a device enters a new alert state.
// If a device is stuck at high temperature, we emit ONE TEMP_HIGH alert —
// not one per tick. When the condition clears, the state resets so the
// next occurrence fires again.
//
// Multiple conditions can be active simultaneously (e.g. TEMP_HIGH + SPIKE).
// Each active condition is tracked and deduped independently.
type anomalyDetector struct {
	tempHigh   float32
	tempLow    float32
	batteryLow float32

	// activeAlerts tracks which alert types are currently firing per device.
	// A type is present ↔ that condition was true on the last tick.
	// On first trigger: emit alert, add to set.
	// While condition persists: no re-emit.
	// When condition clears: remove from set (next trigger re-emits).
	activeAlerts map[string]map[models.AlertType]struct{}
}

// newAnomalyDetector creates a detector with the given threshold values.
func newAnomalyDetector(tempHigh, tempLow, batteryLow float64) *anomalyDetector {
	return &anomalyDetector{
		tempHigh:     float32(tempHigh),
		tempLow:      float32(tempLow),
		batteryLow:   float32(batteryLow),
		activeAlerts: make(map[string]map[models.AlertType]struct{}),
	}
}

// detect evaluates all rules against agg and returns alerts to publish.
// Returns an empty slice when no new alert conditions are detected.
func (a *anomalyDetector) detect(agg *windowAggregate) []models.DeviceAlert {
	now := time.Now().UTC()
	windowSecs := int32(agg.windowEnd.Sub(agg.windowStart).Seconds())
	if windowSecs < 1 {
		windowSecs = 1
	}

	// Evaluate each rule: collect which types are currently firing.
	type candidate struct {
		alertType  models.AlertType
		severity   models.Severity
		metric     float32
		threshold  float32
	}

	var firing []candidate

	if agg.avgTemp > a.tempHigh {
		firing = append(firing, candidate{models.AlertTypeTempHigh, models.SeverityHigh, agg.avgTemp, a.tempHigh})
	}
	if agg.avgTemp < a.tempLow {
		firing = append(firing, candidate{models.AlertTypeTempLow, models.SeverityMedium, agg.avgTemp, a.tempLow})
	}
	if agg.avgBattery < a.batteryLow {
		firing = append(firing, candidate{models.AlertTypeBatteryLow, models.SeverityHigh, agg.avgBattery, a.batteryLow})
	}
	// SPIKE: high intra-window temperature variance (e.g. chaos mode injection)
	spread := agg.maxTemp - agg.minTemp
	if spread > 20.0 {
		firing = append(firing, candidate{models.AlertTypeSpike, models.SeverityCritical, spread, 20.0})
	}

	// Build the set of currently-firing types for this device.
	currentlyFiring := make(map[models.AlertType]struct{}, len(firing))
	for _, c := range firing {
		currentlyFiring[c.alertType] = struct{}{}
	}

	// Ensure the device has an active-alert set.
	if a.activeAlerts[agg.deviceID] == nil {
		a.activeAlerts[agg.deviceID] = make(map[models.AlertType]struct{})
	}
	active := a.activeAlerts[agg.deviceID]

	// Emit only for NEW conditions (not already in active set).
	var alerts []models.DeviceAlert
	for _, c := range firing {
		if _, alreadyActive := active[c.alertType]; alreadyActive {
			continue // same condition, already emitted — skip
		}
		alerts = append(alerts, models.DeviceAlert{
			AlertID:        newUUID(),
			DeviceID:       agg.deviceID,
			AlertType:      c.alertType,
			Severity:       c.severity,
			TriggeredAt:    now,
			MetricValue:    c.metric,
			ThresholdValue: c.threshold,
			WindowSeconds:  windowSecs,
		})
		active[c.alertType] = struct{}{}
	}

	// Clear conditions that are no longer firing so they can re-alert next time.
	for alertType := range active {
		if _, stillFiring := currentlyFiring[alertType]; !stillFiring {
			delete(active, alertType)
		}
	}

	return alerts
}

// newUUID generates a random UUID v4.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
