package main

// generator holds the random number generator and chaos configuration
// used to produce metric values from device state.
//
// Metric generation strategy:
//   - Temperature:    Gaussian noise around device baseline (σ = 2.0°C)
//   - Humidity:       Gaussian noise around 55% (σ = 5%)
//   - Pressure:       Gaussian noise around 1013 hPa (σ = 3 hPa)
//   - Battery level:  Slowly decreasing counter, resets at 100% (models charging)
//   - Chaos mode:     With 5% probability per tick, injects a value that will
//                     cross an anomaly threshold (exercise TEMP_HIGH, BATTERY_LOW etc.)
type generator struct {
	// TODO: *rand.Rand, chaos bool, anomaly thresholds from config
}

// newGenerator creates a generator seeded from crypto/rand for reproducible
// but unpredictable sequences across runs.
func newGenerator() *generator {
	panic("not implemented — implement in Phase 2")
}

// generate produces a DeviceMetric reading for the given device state.
func (g *generator) generate() {
	panic("not implemented — implement in Phase 2")
}
