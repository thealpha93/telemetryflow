package main

// windowStore maintains per-device sliding windows over the last N seconds.
//
// Implementation: each device's window is a ring buffer of timestamped readings.
// On each new reading, entries older than windowSize are evicted from the front.
// Aggregate statistics are computed over the remaining entries.
//
// Concurrency: the processor is single-threaded per partition (one goroutine
// handles all records from a given partition in order). No locking needed
// within a partition. The windowStore is NOT safe for concurrent access.
//
// Memory: at 1000 devices × 60 readings/device = 60,000 entries max.
// Each entry is ~64 bytes → ~3.8 MB total. Well within a single process.
type windowStore struct {
	// TODO: map[deviceID][]windowEntry, windowSize time.Duration
}

// windowEntry is a single data point in a device's sliding window.
type windowEntry struct {
	// TODO: timestamp, temperature, humidity, pressure, batteryLevel
}

// windowAggregate is the computed summary of a device's current window.
// This is what gets published to device.aggregated.
type windowAggregate struct {
	// TODO: deviceID, bucket time, avgTemp, maxTemp, minTemp, avgBattery, sampleCount
}

// newWindowStore creates an empty windowStore with the given window duration.
func newWindowStore() *windowStore {
	panic("not implemented — implement in Phase 3")
}

// push adds a new reading to a device's window and evicts stale entries.
// Returns the current aggregate over the updated window.
func (w *windowStore) push() (*windowAggregate, error) {
	panic("not implemented — implement in Phase 3")
}
