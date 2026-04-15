-- +goose Up
-- Alerts table: current state of each active alert.
-- One row per alert instance. resolved_at is NULL while the alert is active.
-- The sink service upserts into this table on every DeviceAlert event it receives.
--
-- Note: alert_id comes from the DeviceAlert Avro message (a UUID generated
-- by the processor). Using the message's UUID as the PK enables idempotent
-- upserts: replaying the same alert message produces no duplicate rows.
CREATE TABLE alerts (
    alert_id        UUID        PRIMARY KEY,
    device_id       TEXT        NOT NULL REFERENCES devices(device_id),
    alert_type      TEXT        NOT NULL,
    severity        TEXT        NOT NULL,
    triggered_at    TIMESTAMPTZ NOT NULL,
    metric_value    FLOAT       NOT NULL,
    threshold_value FLOAT       NOT NULL,
    window_seconds  INT         NOT NULL,
    resolved_at     TIMESTAMPTZ,           -- NULL = still active
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX alerts_device_id_idx    ON alerts(device_id);
CREATE INDEX alerts_triggered_at_idx ON alerts(triggered_at DESC);
-- Partial index for fast "active alerts" queries (most common query pattern)
CREATE INDEX alerts_active_idx       ON alerts(device_id, triggered_at DESC)
    WHERE resolved_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS alerts_active_idx;
DROP INDEX IF EXISTS alerts_triggered_at_idx;
DROP INDEX IF EXISTS alerts_device_id_idx;
DROP TABLE IF EXISTS alerts;
