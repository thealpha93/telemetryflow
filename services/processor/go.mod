module github.com/telemetryflow/processor

go 1.25.0

replace (
	github.com/telemetryflow/config => ../../config
	github.com/telemetryflow/pkg => ../../pkg
)

require (
	github.com/jackc/pgx/v5 v5.9.1
	github.com/telemetryflow/config v0.0.0-00010101000000-000000000000
	github.com/telemetryflow/pkg v0.0.0-00010101000000-000000000000
	github.com/twmb/franz-go v1.20.7
)

require (
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/hamba/avro/v2 v2.31.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.4 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pierrec/lz4/v4 v4.1.25 // indirect
	github.com/pressly/goose/v3 v3.27.0 // indirect
	github.com/sethvargo/go-retry v0.3.0 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.12.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)
