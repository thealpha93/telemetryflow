-- +goose Up
-- Alert events table: full lifecycle audit trail.
-- Every state transition (TRIGGERED, ACKNOWLEDGED, RESOLVED, ESCALATED)
-- appends a row here. The alerts table holds current state; this table
-- holds the complete history. Together they give both a current-state
-- view and a full audit trail without denormalization.
--
-- This is an append-only table — rows are never updated or deleted.
CREATE TABLE alert_events (
    id          BIGSERIAL   PRIMARY KEY,
    alert_id    UUID        NOT NULL REFERENCES alerts(alert_id),
    device_id   TEXT        NOT NULL,  -- denormalized for query efficiency
    event_type  TEXT        NOT NULL
                            CHECK (event_type IN ('TRIGGERED','ACKNOWLEDGED','RESOLVED','ESCALATED')),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata    JSONB                  -- optional context (e.g. who acknowledged, escalation reason)
);

CREATE INDEX alert_events_alert_id_idx    ON alert_events(alert_id);
CREATE INDEX alert_events_device_id_idx   ON alert_events(device_id);
CREATE INDEX alert_events_occurred_at_idx ON alert_events(occurred_at DESC);

-- +goose Down
DROP INDEX IF EXISTS alert_events_occurred_at_idx;
DROP INDEX IF EXISTS alert_events_device_id_idx;
DROP INDEX IF EXISTS alert_events_alert_id_idx;
DROP TABLE IF EXISTS alert_events;
