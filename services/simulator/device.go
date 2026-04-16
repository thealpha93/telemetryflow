package main

import "math/rand"

// stateType represents which phase a device is currently in.
type stateType int

const (
	stateNormal stateType = iota // sensors reading within expected range
	stateSpike                   // sudden extreme temperature for a few ticks
	stateDrift                   // baseline slowly shifting over many ticks
)

// device models a single IoT sensor's internal state machine.
//
// Each device runs in its own goroutine and calls tick() once per emit
// interval. The state machine produces realistic telemetry patterns:
//
//	normal → spike  (2% chance per tick — sudden temperature extreme)
//	normal → drift  (1% chance per tick — gradual baseline shift)
//	spike  → normal (after 3-7 ticks)
//	drift  → normal (after 20-40 ticks)
//
// Battery drains slowly each tick and resets to 100% to simulate recharging.
type device struct {
	id              string
	locationID      string
	firmwareVersion string

	state          stateType
	stateTicksLeft int  // ticks remaining in current spike/drift
	spikeHigh      bool // true = spike above threshold, false = spike below

	baselineTemp float64 // °C — shifts gradually during drift state
	batteryLevel float32 // % — drains each tick, resets on full discharge
}

// readings holds the raw sensor values produced by a single tick.
// Kept internal — generator.go converts these into a models.DeviceMetric.
type readings struct {
	temperature float64
	humidity    float64
	pressure    float64
	battery     float32
}

// newDevice creates a device with randomised initial conditions so that
// all 1000 devices don't start in identical states.
func newDevice(id, locationID, firmwareVersion string) *device {
	return &device{
		id:              id,
		locationID:      locationID,
		firmwareVersion: firmwareVersion,
		state:           stateNormal,
		// Baseline spread across 20-60°C to model devices in different environments
		// (cold storage, outdoor, server room, etc.)
		baselineTemp: 20.0 + rand.Float64()*40.0,
		// Start at a random battery level so devices don't all hit low-battery
		// at the same time during a test run.
		batteryLevel: float32(60.0 + rand.Float64()*40.0),
	}
}

// tick advances the state machine by one step and returns the current
// sensor readings. Called once per emit interval by the device goroutine.
func (d *device) tick() readings {
	d.advanceState()
	d.drainBattery()
	return d.currentReadings()
}

// advanceState handles state transitions and in-state behaviour.
func (d *device) advanceState() {
	switch d.state {
	case stateNormal:
		r := rand.Float64()
		switch {
		case r < 0.02: // 2% → spike
			d.state = stateSpike
			d.stateTicksLeft = 3 + rand.Intn(5)   // 3-7 ticks
			d.spikeHigh = rand.Float64() < 0.7     // 70% spike high, 30% spike low
		case r < 0.03: // 1% → drift
			d.state = stateDrift
			d.stateTicksLeft = 20 + rand.Intn(21) // 20-40 ticks
		}

	case stateSpike, stateDrift:
		d.stateTicksLeft--
		if d.stateTicksLeft <= 0 {
			d.state = stateNormal
		}
		// During drift, nudge the baseline each tick (±0.25°C)
		if d.state == stateDrift {
			d.baselineTemp += (rand.Float64() - 0.5) * 0.5
		}
	}
}

// drainBattery decreases battery by a small random amount each tick.
// When it hits 0% it resets to 100% — simulating a recharge cycle.
func (d *device) drainBattery() {
	drain := float32(0.01 + rand.Float64()*0.02) // 0.01-0.03% per tick
	d.batteryLevel -= drain
	if d.batteryLevel < 0 {
		d.batteryLevel = 100.0
	}
}

// currentReadings produces sensor values appropriate for the current state.
// Normal and drift states apply Gaussian noise around the baseline.
// Spike states jump to extreme values designed to cross anomaly thresholds.
func (d *device) currentReadings() readings {
	var temp float64
	switch d.state {
	case stateNormal, stateDrift:
		temp = d.baselineTemp + rand.NormFloat64()*2.0 // σ = 2°C
	case stateSpike:
		if d.spikeHigh {
			temp = 90.0 + rand.NormFloat64()*3.0 // above PROCESSOR_ANOMALY_TEMP_HIGH (85°C)
		} else {
			temp = -15.0 + rand.NormFloat64()*3.0 // below PROCESSOR_ANOMALY_TEMP_LOW (-10°C)
		}
	}

	return readings{
		temperature: temp,
		humidity:    55.0 + rand.NormFloat64()*5.0,   // 55% ±5%
		pressure:    1013.0 + rand.NormFloat64()*3.0, // 1013 hPa ±3
		battery:     d.batteryLevel,
	}
}
