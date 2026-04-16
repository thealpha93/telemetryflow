package main

import (
	"time"

	"github.com/telemetryflow/pkg/models"
)

// windowStore maintains per-device sliding windows over the last N seconds.
//
// Each device has a slice of windowEntry values ordered by timestamp.
// On push, entries older than windowSize relative to the incoming reading
// are evicted from the front. Aggregates are then computed in one pass.
//
// Memory: 1000 devices × 60 readings × ~24 bytes/entry ≈ 1.4 MB. Fine.
//
// NOT safe for concurrent access — one goroutine handles a partition's
// records in order, so no locking is needed.
type windowStore struct {
	windows    map[string][]windowEntry
	windowSize time.Duration
}

// windowEntry holds only the fields needed for aggregate computation.
type windowEntry struct {
	timestamp    time.Time
	temperature  float32
	batteryLevel float32
}

// windowAggregate is the computed summary over a device's current window.
// The processorConsumer enriches this with device metadata before publishing.
type windowAggregate struct {
	deviceID    string
	windowStart time.Time // timestamp of the oldest retained entry
	windowEnd   time.Time // timestamp of the just-pushed reading
	avgTemp     float32
	maxTemp     float32
	minTemp     float32
	avgBattery  float32
	sampleCount int
}

// newWindowStore creates an empty store with the given window duration.
func newWindowStore(windowSize time.Duration) *windowStore {
	return &windowStore{
		windows:    make(map[string][]windowEntry),
		windowSize: windowSize,
	}
}

// push adds metric to the device's window, evicts stale entries, and
// returns the updated aggregate. Always returns a non-nil aggregate —
// the new entry is appended before eviction so the window is never empty.
func (w *windowStore) push(metric models.DeviceMetric) (*windowAggregate, error) {
	entries := w.windows[metric.DeviceID]

	// Evict entries older than (metric.Timestamp - windowSize).
	// Because records arrive roughly in order, this is usually a no-op or
	// trims just a few entries from the front — amortised O(1).
	cutoff := metric.Timestamp.Add(-w.windowSize)
	trim := 0
	for trim < len(entries) && entries[trim].timestamp.Before(cutoff) {
		trim++
	}

	// Keep in-window entries and append the new one.
	entries = append(entries[trim:], windowEntry{
		timestamp:    metric.Timestamp,
		temperature:  metric.Temperature,
		batteryLevel: metric.BatteryLevel,
	})
	w.windows[metric.DeviceID] = entries

	// Single-pass min/max/sum.
	var sumTemp, sumBattery float32
	maxTemp := entries[0].temperature
	minTemp := entries[0].temperature

	for _, e := range entries {
		sumTemp += e.temperature
		sumBattery += e.batteryLevel
		if e.temperature > maxTemp {
			maxTemp = e.temperature
		}
		if e.temperature < minTemp {
			minTemp = e.temperature
		}
	}

	n := float32(len(entries))
	return &windowAggregate{
		deviceID:    metric.DeviceID,
		windowStart: entries[0].timestamp,
		windowEnd:   metric.Timestamp,
		avgTemp:     sumTemp / n,
		maxTemp:     maxTemp,
		minTemp:     minTemp,
		avgBattery:  sumBattery / n,
		sampleCount: len(entries),
	}, nil
}
