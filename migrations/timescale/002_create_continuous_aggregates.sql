-- +goose Up
-- Continuous aggregates: TimescaleDB materializes these automatically as
-- new data arrives. No manual refresh needed. Queries against these views
-- are fast because they read pre-computed results, not raw rows.
--
-- Two granularities:
--   device_metrics_1min  — 1-minute buckets (high resolution, recent data)
--   device_metrics_5min  — 5-minute buckets (for longer time range queries)

CREATE MATERIALIZED VIEW device_metrics_1min
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 minute', time) AS bucket,
    device_id,
    AVG(temperature)              AS avg_temp,
    MAX(temperature)              AS max_temp,
    MIN(temperature)              AS min_temp,
    AVG(battery_level)            AS avg_battery,
    COUNT(*)                      AS sample_count
FROM device_metrics
GROUP BY bucket, device_id
WITH NO DATA;  -- populate on first refresh, not at creation time

-- Automatically refresh the 1-min view as new data arrives
SELECT add_continuous_aggregate_policy('device_metrics_1min',
    start_offset => INTERVAL '10 minutes',
    end_offset   => INTERVAL '1 minute',
    schedule_interval => INTERVAL '1 minute');

CREATE MATERIALIZED VIEW device_metrics_5min
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('5 minutes', time)                                  AS bucket,
    device_id,
    AVG(temperature)                                                AS avg_temp,
    MAX(temperature)                                                AS max_temp,
    MIN(temperature)                                                AS min_temp,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY temperature)       AS p95_temp,
    AVG(battery_level)                                              AS avg_battery
FROM device_metrics
GROUP BY bucket, device_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('device_metrics_5min',
    start_offset => INTERVAL '1 hour',
    end_offset   => INTERVAL '5 minutes',
    schedule_interval => INTERVAL '5 minutes');

-- +goose Down
SELECT remove_continuous_aggregate_policy('device_metrics_5min', if_exists => true);
DROP MATERIALIZED VIEW IF EXISTS device_metrics_5min;

SELECT remove_continuous_aggregate_policy('device_metrics_1min', if_exists => true);
DROP MATERIALIZED VIEW IF EXISTS device_metrics_1min;
