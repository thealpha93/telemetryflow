package main

// deviceState represents the internal state machine of a single simulated device.
//
// Each device cycles through states to produce realistic telemetry:
//
//	normal ──► spike  (random chance per tick in normal state)
//	spike  ──► normal (after spike duration expires)
//	normal ──► drift  (random chance, models gradual sensor degradation)
//	drift  ──► normal (after drift duration expires)
//
// In normal state: temperature drifts around a baseline using Gaussian noise.
// In spike state:  temperature jumps to a high/low extreme for a short duration.
// In drift state:  baseline slowly shifts up or down each tick.
//
// This produces data that exercises all anomaly detection rules in the processor
// without a physical device.
type deviceState struct {
	// TODO: device_id, location_id, firmware_version, current state,
	//       baseline temperature, tick counter
}

// newDevice initialises a device with the given ID and a random baseline.
func newDevice() *deviceState {
	panic("not implemented — implement in Phase 2")
}

// next advances the state machine by one tick and returns the next metric.
// This is the only method callers need — the state transitions are internal.
func (d *deviceState) next() {
	panic("not implemented — implement in Phase 2")
}
