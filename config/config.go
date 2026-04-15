// Package config is the single source of truth for all TelemetryFlow
// environment variable configuration. Every service imports this package
// and calls Load() on startup. If any required variable is missing the
// process exits immediately with a clear error — no silent defaults.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

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
// Returns a descriptive error listing ALL missing variables at once so
// the developer doesn't have to fix them one by one.
func Load() (*Config, error) {
	// Load .env into the process environment if the file exists.
	// Silently ignored if the file is absent — in production, env vars
	// are injected by the platform (Kubernetes, systemd, etc.).
	_ = godotenv.Load()

	var missing []string

	// requireStr reads a required env var. Appends to missing if absent.
	requireStr := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	// optStr reads an optional env var, returning def if absent.
	optStr := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return def
	}

	// optInt reads an optional integer env var, returning def if absent or unparseable.
	optInt := func(key string, def int) int {
		v := os.Getenv(key)
		if v == "" {
			return def
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			missing = append(missing, fmt.Sprintf("%s (must be an integer, got %q)", key, v))
			return def
		}
		return n
	}

	// optFloat reads an optional float64 env var, returning def if absent or unparseable.
	optFloat := func(key string, def float64) float64 {
		v := os.Getenv(key)
		if v == "" {
			return def
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			missing = append(missing, fmt.Sprintf("%s (must be a float, got %q)", key, v))
			return def
		}
		return f
	}

	// optBool reads an optional boolean env var, returning def if absent or unparseable.
	optBool := func(key string, def bool) bool {
		v := os.Getenv(key)
		if v == "" {
			return def
		}
		b, err := strconv.ParseBool(v)
		if err != nil {
			missing = append(missing, fmt.Sprintf("%s (must be true/false, got %q)", key, v))
			return def
		}
		return b
	}

	cfg := &Config{
		Kafka: KafkaConfig{
			BootstrapServers:        requireStr("KAFKA_BOOTSTRAP_SERVERS"),
			APIKey:                  requireStr("KAFKA_API_KEY"),
			APISecret:               requireStr("KAFKA_API_SECRET"),
			SchemaRegistryURL:       requireStr("SCHEMA_REGISTRY_URL"),
			SchemaRegistryAPIKey:    requireStr("SCHEMA_REGISTRY_API_KEY"),
			SchemaRegistryAPISecret: requireStr("SCHEMA_REGISTRY_API_SECRET"),
		},
		Postgres: PostgresConfig{
			DSN: requireStr("POSTGRES_DSN"),
		},
		Timescale: TimescaleConfig{
			DSN: requireStr("TIMESCALE_DSN"),
		},
		Simulator: SimulatorConfig{
			DeviceCount:    optInt("SIMULATOR_DEVICE_COUNT", 1000),
			EmitIntervalMS: optInt("SIMULATOR_EMIT_INTERVAL_MS", 1000),
			ChaosMode:      optBool("SIMULATOR_CHAOS_MODE", false),
		},
		Ingestor: IngestorConfig{
			BatchSize:       optInt("INGESTOR_BATCH_SIZE", 500),
			FlushIntervalMS: optInt("INGESTOR_FLUSH_INTERVAL_MS", 500),
			ConsumerGroup:   optStr("INGESTOR_CONSUMER_GROUP", "telemetryflow-ingestor"),
		},
		Processor: ProcessorConfig{
			ConsumerGroup:          optStr("PROCESSOR_CONSUMER_GROUP", "telemetryflow-processor"),
			WindowSizeSeconds:      optInt("PROCESSOR_WINDOW_SIZE_SECONDS", 60),
			AnomalyTempHigh:        optFloat("PROCESSOR_ANOMALY_TEMP_HIGH", 85.0),
			AnomalyTempLow:         optFloat("PROCESSOR_ANOMALY_TEMP_LOW", -10.0),
			AnomalyBatteryLow:      optFloat("PROCESSOR_ANOMALY_BATTERY_LOW", 10.0),
			EnrichmentCacheTTLSecs: optInt("PROCESSOR_ENRICHMENT_CACHE_TTL_SECONDS", 300),
		},
		Sink: SinkConfig{
			ConsumerGroup:   optStr("SINK_CONSUMER_GROUP", "telemetryflow-sink"),
			BatchSize:       optInt("SINK_BATCH_SIZE", 100),
			FlushIntervalMS: optInt("SINK_FLUSH_INTERVAL_MS", 1000),
		},
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf(
			"missing or invalid environment variables:\n  %s\n\nCopy .env.example to .env and fill in the values.",
			strings.Join(missing, "\n  "),
		)
	}

	return cfg, nil
}
