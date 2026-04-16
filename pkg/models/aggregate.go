package models

import "time"

// DeviceAggregate is the output of the stream processor's sliding window
// computation for a single device. Published to device.aggregated.
//
// Data path: Processor → device.aggregated (Kafka) → Sink → PostgreSQL
//
// The avro struct tags must match device_aggregate.avsc exactly.
type DeviceAggregate struct {
	DeviceID    string    `avro:"device_id"`
	LocationID  string    `avro:"location_id"`
	WindowStart time.Time `avro:"window_start"`
	WindowEnd   time.Time `avro:"window_end"`
	AvgTemp     float32   `avro:"avg_temp"`
	MaxTemp     float32   `avro:"max_temp"`
	MinTemp     float32   `avro:"min_temp"`
	AvgBattery  float32   `avro:"avg_battery"`
	SampleCount int32     `avro:"sample_count"`
}
