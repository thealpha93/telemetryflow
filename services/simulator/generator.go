package main

import (
	"math/rand"
	"time"

	"github.com/telemetryflow/pkg/models"
)

// generator converts raw device readings into DeviceMetric domain objects.
// It is stateless — all state lives in the device struct.
//
// Chaos mode (SIMULATOR_CHAOS_MODE=true) adds a 5% per-tick chance of
// injecting an extreme value regardless of the device's current state.
// This exercises the processor's anomaly detection without waiting for the
// state machine to naturally enter a spike.
type generator struct {
	chaosMode bool
}

// newGenerator creates a generator with the given chaos mode setting.
func newGenerator(chaosMode bool) *generator {
	return &generator{chaosMode: chaosMode}
}

// generate ticks the device state machine and returns a DeviceMetric.
// Timestamp is set to UTC now — the consumer side stores it as-is in TimescaleDB.
func (g *generator) generate(d *device) models.DeviceMetric {
	r := d.tick()

	// Chaos mode: 5% chance per tick to inject a TEMP_HIGH anomaly
	// regardless of the device's natural state.
	if g.chaosMode && rand.Float64() < 0.05 {
		r.temperature = 95.0 + rand.NormFloat64()*2.0
	}

	return models.DeviceMetric{
		DeviceID:        d.id,
		Timestamp:       time.Now().UTC(),
		Temperature:     float32(r.temperature),
		Humidity:        float32(r.humidity),
		Pressure:        float32(r.pressure),
		BatteryLevel:    r.battery,
		FirmwareVersion: d.firmwareVersion,
		LocationID:      d.locationID,
	}
}
