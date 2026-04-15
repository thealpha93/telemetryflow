// Package config is the single source of truth for all TelemetryFlow
// environment variable configuration. Every service imports this package
// and calls Load() on startup. If any required variable is missing the
// process exits immediately with a clear error — no silent defaults.
package config

// Config is the root configuration object. Each service only reads the
// sub-struct it cares about (e.g. the simulator only needs Kafka and Simulator).
type Config struct {
	Kafka     KafkaConfig
	Postgres  PostgresConfig
	Timescale TimescaleConfig
	Simulator SimulatorConfig
	Ingestor  IngestorConfig
	Processor ProcessorConfig
	Sink      SinkConfig
}

// KafkaConfig holds credentials for the Confluent Cloud cluster and
// the embedded Schema Registry. Both share the same env-var prefix.
type KafkaConfig struct {
	BootstrapServers        string // KAFKA_BOOTSTRAP_SERVERS
	APIKey                  string // KAFKA_API_KEY
	APISecret               string // KAFKA_API_SECRET
	SchemaRegistryURL       string // SCHEMA_REGISTRY_URL
	SchemaRegistryAPIKey    string // SCHEMA_REGISTRY_API_KEY
	SchemaRegistryAPISecret string // SCHEMA_REGISTRY_API_SECRET
}

// PostgresConfig holds the DSN for the local PostgreSQL instance
// (device registry, alerts, DLQ tables).
type PostgresConfig struct {
	DSN string // POSTGRES_DSN
}

// TimescaleConfig holds the DSN for Timescale Cloud
// (raw device_metrics hypertable and continuous aggregates).
type TimescaleConfig struct {
	DSN string // TIMESCALE_DSN
}

// SimulatorConfig controls how many devices are simulated and at what rate.
type SimulatorConfig struct {
	DeviceCount    int  // SIMULATOR_DEVICE_COUNT     (default: 1000)
	EmitIntervalMS int  // SIMULATOR_EMIT_INTERVAL_MS (default: 1000)
	ChaosMode      bool // SIMULATOR_CHAOS_MODE       (default: false)
}

// IngestorConfig controls batching behaviour for the TimescaleDB writer.
// A batch is flushed when it reaches BatchSize records OR FlushIntervalMS
// milliseconds have elapsed — whichever comes first (ADR-005).
type IngestorConfig struct {
	BatchSize       int    // INGESTOR_BATCH_SIZE        (default: 500)
	FlushIntervalMS int    // INGESTOR_FLUSH_INTERVAL_MS (default: 500)
	ConsumerGroup   string // INGESTOR_CONSUMER_GROUP
}

// ProcessorConfig controls the stream processor's sliding window and
// anomaly detection thresholds.
type ProcessorConfig struct {
	ConsumerGroup          string  // PROCESSOR_CONSUMER_GROUP
	WindowSizeSeconds      int     // PROCESSOR_WINDOW_SIZE_SECONDS       (default: 60)
	AnomalyTempHigh        float64 // PROCESSOR_ANOMALY_TEMP_HIGH         (default: 85.0)
	AnomalyTempLow         float64 // PROCESSOR_ANOMALY_TEMP_LOW          (default: -10.0)
	AnomalyBatteryLow      float64 // PROCESSOR_ANOMALY_BATTERY_LOW       (default: 10.0)
	EnrichmentCacheTTLSecs int     // PROCESSOR_ENRICHMENT_CACHE_TTL_SECONDS (default: 300)
}

// SinkConfig controls batching behaviour for the PostgreSQL/TimescaleDB writers
// in the sink service.
type SinkConfig struct {
	ConsumerGroup   string // SINK_CONSUMER_GROUP
	BatchSize       int    // SINK_BATCH_SIZE        (default: 100)
	FlushIntervalMS int    // SINK_FLUSH_INTERVAL_MS (default: 1000)
}

// Load reads all configuration from environment variables.
// It loads a .env file first if one is present (local development only).
// Returns a descriptive error if any required variable is missing or unparseable.
func Load() (*Config, error) {
	panic("not implemented — implement in Phase 1")
}
