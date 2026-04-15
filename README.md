# TelemetryFlow

A production-grade IoT telemetry pipeline built in Go. Ingests high-frequency device metrics from 1000+ simulated sensors, processes them in real-time with sliding window aggregations, detects anomalies, and persists results to time-series and relational databases.

Built to learn Kafka and Go by building something real — not a toy app. Every design decision reflects production standards: fault tolerance, explicit error handling, structured observability, graceful shutdown, and environment-based configuration.

---

## Architecture

```
Device Simulator (Go)
    │  franz-go producer · Avro + Schema Registry · idempotent
    ▼
Kafka Topic: device.metrics  (12 partitions, keyed by device_id)
    │
    ├─────────────────────────────────────┐
    ▼                                     ▼
Ingestor Service (Go)         Stream Processor Service (Go)
  · exactly-once writes          · sliding window aggregations
  · batched inserts (500/500ms)  · anomaly detection
  · DLQ on failure               · device registry enrichment
    │                                     │
    ▼                             ┌───────┴───────┐
TimescaleDB Cloud                 ▼               ▼
(raw device_metrics)    device.aggregated    device.alerts
                                  │               │
                                  └───────┬───────┘
                                          ▼
                                  Sink Service (Go)
                                    · idempotent upserts
                                    · DLQ handling
                                    · per-partition error isolation
                                          │
                             ┌────────────┴────────────┐
                             ▼                         ▼
                       TimescaleDB Cloud          PostgreSQL (local)
                       (aggregated metrics)  (alerts · DLQ · device registry)
```

Services communicate **only through Kafka** — no direct service-to-service calls. This means each service can be deployed, scaled, and restarted independently.

---

## Tech Stack

| Component | Library / Service | Why |
|---|---|---|
| Kafka client | `github.com/twmb/franz-go` | Pure Go, no CGO, full protocol control |
| PostgreSQL driver | `github.com/jackc/pgx/v5` | Native binary protocol, pgxpool for concurrency |
| Avro serialization | `github.com/hamba/avro/v2` | Pure Go, Confluent wire format compatible |
| Schema Registry | Confluent Cloud (managed) | Schema enforcement + compatibility checks |
| Migrations | `github.com/pressly/goose/v3` | SQL-based, tracked, idempotent |
| Config | `github.com/joho/godotenv` + `os.Getenv` | 12-factor, no third-party config library |
| Logging | `log/slog` (stdlib) | Structured JSON, zero dependencies |
| Message broker | Confluent Cloud (managed) | No local Kafka — real cloud infrastructure |
| Time-series DB | Timescale Cloud (managed) | Hypertables + continuous aggregates |
| Relational DB | PostgreSQL (local via Docker) | Device registry, alerts, DLQ |
| Testing | `testify` + `testcontainers-go` | Assertions + real Postgres for integration tests |

---

## Repository Structure

```
telemetryflow/
├── go.work                          ← Go workspace (links all modules)
├── docker-compose.yml               ← Local PostgreSQL only
├── Makefile                         ← All dev commands (run `make help`)
├── .env.example                     ← Environment variable template
│
├── config/
│   └── config.go                    ← Centralised env config — all services import this
│
├── pkg/                             ← Shared libraries (no business logic)
│   ├── kafka/
│   │   ├── topics.go                ← Topic + consumer group constants
│   │   ├── producer.go              ← Idempotent franz-go producer wrapper
│   │   ├── consumer.go              ← Manual-commit consumer wrapper
│   │   ├── transactional.go         ← Transactional producer (EOS for ingestor)
│   │   └── dlq.go                   ← Dead-letter queue publisher
│   ├── schema/
│   │   ├── registry.go              ← Confluent Schema Registry client
│   │   ├── avro.go                  ← Avro serializer / deserializer
│   │   └── schemas/                 ← device_metric.avsc · device_alert.avsc
│   ├── models/
│   │   ├── metric.go                ← DeviceMetric domain type
│   │   ├── alert.go                 ← DeviceAlert domain type + enums
│   │   └── device.go                ← Device registry domain type
│   ├── db/
│   │   ├── postgres.go              ← pgxpool factory for PostgreSQL
│   │   ├── timescale.go             ← pgxpool factory for TimescaleDB
│   │   └── migrations.go            ← goose migration runner
│   └── logger/
│       └── logger.go                ← Structured JSON logger (slog)
│
├── services/
│   ├── simulator/                   ← Produces device.metrics
│   │   ├── main.go
│   │   ├── device.go                ← Device state machine (normal/spike/drift)
│   │   └── generator.go             ← Metric value generation with Gaussian noise
│   ├── ingestor/                    ← Consumes device.metrics → TimescaleDB
│   │   ├── main.go
│   │   ├── consumer.go              ← Poll → batch → flush loop
│   │   └── writer.go                ← Batched TimescaleDB writer (COPY protocol)
│   ├── processor/                   ← Sliding windows · anomaly detection · enrichment
│   │   ├── main.go
│   │   ├── consumer.go              ← Poll → process → publish loop
│   │   ├── window.go                ← Per-device sliding window state
│   │   ├── anomaly.go               ← TEMP_HIGH / TEMP_LOW / BATTERY_LOW / SPIKE rules
│   │   ├── enricher.go              ← Device registry cache (TTL-based)
│   │   └── publisher.go             ← Produces to device.aggregated + device.alerts
│   └── sink/                        ← Consumes device.aggregated + device.alerts → DB
│       ├── main.go
│       ├── consumer.go              ← Fan-out to two writer pipelines
│       └── writer.go                ← Idempotent upsert writers
│
├── migrations/
│   ├── postgres/                    ← 001_devices · 002_alerts · 003_alert_events · 004_dlq
│   └── timescale/                   ← 001_device_metrics · 002_continuous_aggregates
│
└── scripts/
    ├── create-topics.sh             ← Creates all Confluent Cloud topics
    └── seed-devices.sql             ← Seeds 1000 test devices
```

---

## Kafka Topics

| Topic | Partitions | Retention | Key | Format |
|---|---|---|---|---|
| `device.metrics` | 12 | 7 days | `device_id` | Avro |
| `device.aggregated` | 12 | 30 days | `device_id` | Avro |
| `device.alerts` | 6 | 90 days | `device_id` | Avro |
| `device.metrics.dlq` | 3 | 30 days | `device_id` | Avro |
| `device.alerts.dlq` | 3 | 30 days | `device_id` | Avro |

All topics partition by `device_id` — guaranteeing per-device ordering and co-locating all messages from the same device on the same partition (critical for stateful window processing).

`device.alerts` is a **public contract** consumed by an external notification service. Schema changes must always be backward-compatible — never remove or rename fields or enum values.

---

## Prerequisites

| Tool | Purpose | Install |
|---|---|---|
| Go 1.23+ | Build and run services | [go.dev](https://go.dev/dl/) |
| PostgreSQL | Local database (device registry, alerts, DLQ) | Already installed locally |
| goose | Run database migrations | `go install github.com/pressly/goose/v3/cmd/goose@latest` |
| Confluent CLI | Create Kafka topics | `brew install confluentinc/tap/cli` |
| Confluent Cloud account | Managed Kafka + Schema Registry | [confluent.io](https://confluent.io) (free tier) |
| Timescale Cloud account | Managed TimescaleDB | [timescale.com](https://timescale.com) (free trial) |

---

## Quick Start

### 1. Configure credentials

```bash
cp .env.example .env
# Edit .env and fill in:
#   KAFKA_BOOTSTRAP_SERVERS, KAFKA_API_KEY, KAFKA_API_SECRET
#   SCHEMA_REGISTRY_URL, SCHEMA_REGISTRY_API_KEY, SCHEMA_REGISTRY_API_SECRET
#   TIMESCALE_DSN
#   CONFLUENT_CLUSTER_ID
```

### 2. Create the local database

```bash
make create-db    # runs: createdb telemetryflow
```

### 3. Apply database migrations

```bash
make migrate-postgres    # creates devices, alerts, alert_events, dead_letter_queue
make migrate-timescale   # creates device_metrics hypertable + continuous aggregates
```

### 4. Create Confluent Cloud topics

```bash
make create-topics       # runs scripts/create-topics.sh
```

### 5. Seed test devices

```bash
make seed                # inserts 1000 test devices into PostgreSQL
```

### 6. Run the pipeline (one terminal per service)

```bash
make run-simulator       # starts emitting metrics at 1000 devices × 1 msg/sec
make run-ingestor        # consumes device.metrics → TimescaleDB
make run-processor       # anomaly detection → device.aggregated + device.alerts
make run-sink            # persists aggregated metrics + alerts to databases
```

All services log structured JSON to stdout. `^C` triggers graceful shutdown on each.

---

## All Make Targets

```
make help                # full list with descriptions
make build               # build all four service binaries into bin/
make test                # run all tests
make test-race           # run tests with race detector
make vet                 # go vet across all modules
make tidy                # go mod tidy + go work sync
make create-db           # create the local telemetryflow database
make drop-db             # drop the local database (destructive)
make migrate-postgres-status   # show which migrations have been applied
make migrate-timescale-status
```

---

## Environment Variables

All configuration comes from environment variables — no config files. The `.env` file is loaded automatically in development (gitignored; never commit it).

```env
# Kafka (Confluent Cloud)
KAFKA_BOOTSTRAP_SERVERS=pkc-xxxxx.region.confluent.cloud:9092
KAFKA_API_KEY=
KAFKA_API_SECRET=

# Schema Registry (Confluent Cloud)
SCHEMA_REGISTRY_URL=https://psrc-xxxxx.region.confluent.cloud
SCHEMA_REGISTRY_API_KEY=
SCHEMA_REGISTRY_API_SECRET=

# PostgreSQL (local)
POSTGRES_DSN=postgres://telemetryflow:telemetryflow@localhost:5432/telemetryflow?sslmode=disable

# TimescaleDB (Timescale Cloud)
TIMESCALE_DSN=postgres://user:password@host.timescaledb.io:5432/tsdb?sslmode=require

# Simulator
SIMULATOR_DEVICE_COUNT=1000         # number of simulated devices
SIMULATOR_EMIT_INTERVAL_MS=1000     # emit interval per device
SIMULATOR_CHAOS_MODE=false          # inject random anomalies

# Ingestor
INGESTOR_BATCH_SIZE=500             # records per TimescaleDB flush
INGESTOR_FLUSH_INTERVAL_MS=500      # max ms between flushes
INGESTOR_CONSUMER_GROUP=telemetryflow-ingestor

# Processor
PROCESSOR_CONSUMER_GROUP=telemetryflow-processor
PROCESSOR_WINDOW_SIZE_SECONDS=60
PROCESSOR_ANOMALY_TEMP_HIGH=85.0
PROCESSOR_ANOMALY_TEMP_LOW=-10.0
PROCESSOR_ANOMALY_BATTERY_LOW=10.0
PROCESSOR_ENRICHMENT_CACHE_TTL_SECONDS=300

# Sink
SINK_CONSUMER_GROUP=telemetryflow-sink
SINK_BATCH_SIZE=100
SINK_FLUSH_INTERVAL_MS=1000
```

---

## Scaling

Each service is an independent binary that can run on separate machines. Services coordinate exclusively through Kafka — no direct service-to-service networking.

| Service | How to scale | Limit |
|---|---|---|
| Simulator | Run multiple instances with non-overlapping device ID ranges | Unlimited |
| Ingestor | Run N instances in the same consumer group | 12 (= partition count) |
| Processor | Run N instances in the same consumer group | 12 (= partition count) |
| Sink | Run N instances in the same consumer group | 12 (= partition count) |

Kafka distributes partitions across instances automatically using the cooperative sticky assignor.

---

## Key Architectural Decisions

| ADR | Decision | Reason |
|---|---|---|
| ADR-001 | franz-go over confluent-kafka-go | Pure Go, no CGO, full protocol visibility |
| ADR-002 | Manual offset commits everywhere | Auto-commit causes data loss on crash; manual commit after DB write gives at-least-once with known semantics |
| ADR-003 | Idempotent producer always on | Prevents duplicate messages on producer retry at zero cost |
| ADR-004 | Exactly-once in ingestor only | Full EOS is expensive; other services use DB-level upserts which are sufficient |
| ADR-005 | Batched writes to TimescaleDB | 1000 row-by-row inserts/sec would saturate connections; batching reduces to ~2 roundtrips/sec |
| ADR-006 | Cooperative sticky rebalancing | Eager rebalancing pauses all partitions; cooperative only moves what needs to move — critical for stateful window processing |
| ADR-007 | DLQ for all unprocessable messages | Poison pills must not block a partition; DLQ allows inspection and replay without redeployment |
| ADR-008 | Environment-based config | 12-factor compliance; credentials never in committed files |
| ADR-009 | pgxpool everywhere | Single pgx.Conn serializes concurrent DB ops; pool is required for correctness |
| ADR-010 | log/slog for logging | Structured, performant, zero dependencies — available in stdlib since Go 1.21 |

---

## Build Phases

### Phase 1 — Foundation (in progress)
Shared infrastructure: `config.Load()`, structured logger, pgxpool factories, goose migration runner, franz-go producer/consumer wrappers, Schema Registry client, Avro serializer/deserializer.

### Phase 2 — Simulator + Ingestor
First real data flowing end-to-end: simulated devices emitting Avro metrics → Kafka → TimescaleDB with exactly-once semantics.

### Phase 3 — Stream Processor
Sliding window aggregations, anomaly detection (TEMP\_HIGH, TEMP\_LOW, BATTERY\_LOW, SPIKE), device enrichment from PostgreSQL cache.

### Phase 4 — Sink
Idempotent persistence of aggregated metrics (TimescaleDB) and alerts (PostgreSQL alerts + alert\_events tables).

### Phase 5 — Production Hardening
Schema evolution, multi-instance processor with cooperative rebalancing, load testing at 10k devices, chaos testing (kill a service mid-flight, verify no data loss).

---

## Database Schema Summary

**PostgreSQL (local)**
- `devices` — device registry (source of truth for device metadata)
- `alerts` — current alert state, one row per active alert
- `alert_events` — full alert lifecycle audit trail (append-only)
- `dead_letter_queue` — failed Kafka messages awaiting inspection/replay

**TimescaleDB Cloud**
- `device_metrics` — raw hypertable (permanent storage, SQL-queryable)
- `device_metrics_1min` — continuous aggregate (1-minute rollups, auto-refreshed)
- `device_metrics_5min` — continuous aggregate (5-minute rollups with p95 temperature)

---

## Code Conventions

- **Error wrapping**: always `fmt.Errorf("component: operation: %w", err)` — never bare `return err`
- **Context**: every function that does I/O takes `context.Context` as its first argument
- **No `log.Fatal` outside `main()`**: prevents deferred cleanup from running
- **No hardcoded topic names**: use constants from `pkg/kafka/topics.go`
- **No `time.Sleep` in loops**: use `time.Ticker` with a `select`
- **No shared `pgx.Conn`**: always use `pgxpool.Pool`
