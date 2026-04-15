package main

// anomalyDetector evaluates a set of rules against a window aggregate and
// returns any alerts that should be emitted.
//
// Rules (all configurable via env vars):
//
//	TEMP_HIGH    avg_temp > PROCESSOR_ANOMALY_TEMP_HIGH      → severity HIGH
//	TEMP_LOW     avg_temp < PROCESSOR_ANOMALY_TEMP_LOW       → severity MEDIUM
//	BATTERY_LOW  avg_battery < PROCESSOR_ANOMALY_BATTERY_LOW → severity HIGH
//	SPIKE        max_temp - min_temp > 20°C in window        → severity CRITICAL
//
// Alert deduplication: the detector tracks the last alert type per device.
// It does NOT re-emit an alert for the same condition on every tick — only
// when the condition first triggers. This prevents flooding device.alerts
// with duplicate events while a device is stuck in an anomalous state.
//
// The OFFLINE alert type is not detected here — it requires a separate
// heartbeat timeout mechanism (Phase 5 work).
type anomalyDetector struct {
	// TODO: thresholds from config, map[deviceID]lastAlertType for dedup
}

// newAnomalyDetector creates a detector with the configured thresholds.
func newAnomalyDetector() *anomalyDetector {
	panic("not implemented — implement in Phase 3")
}

// detect evaluates all rules against the given aggregate.
// Returns a slice of alerts to emit (may be empty).
func (a *anomalyDetector) detect() {
	panic("not implemented — implement in Phase 3")
}
