-- +goose Up
-- Raw device metrics hypertable.
-- TimescaleDB partitions this automatically by time (chunks of 7 days by default).
-- Queries scoped to a time range only scan the relevant chunks — much faster
-- than a regular PostgreSQL table at IoT scale.
--
-- No PRIMARY KEY constraint: TimescaleDB hypertables do not support primary keys
-- that don't include the partitioning column (time). Deduplication is handled
-- at the producer level (idempotent producer + exactly-once ingestor, ADR-003/004).
CREATE TABLE device_metrics (
    time             TIMESTAMPTZ NOT NULL,
    device_id        TEXT        NOT NULL,
    temperature      FLOAT,
    humidity         FLOAT,
    pressure         FLOAT,
    battery_level    FLOAT,
    firmware_version TEXT,
    location_id      TEXT
);

-- Convert to hypertable, partitioned by time
SELECT create_hypertable('device_metrics', 'time');

-- Composite index for per-device time-range queries (most common access pattern)
CREATE INDEX device_metrics_device_id_time_idx
    ON device_metrics(device_id, time DESC);

-- +goose Down
DROP INDEX IF EXISTS device_metrics_device_id_time_idx;
DROP TABLE IF EXISTS device_metrics;
