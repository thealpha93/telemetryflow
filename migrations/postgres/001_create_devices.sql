-- +goose Up
-- Device registry: source of truth for all known IoT devices.
-- The processor's enricher cache is populated from this table.
-- The alerts table references this table via FK to enforce referential integrity.
CREATE TABLE devices (
    device_id        TEXT PRIMARY KEY,
    name             TEXT        NOT NULL,
    location_id      TEXT        NOT NULL,
    firmware_version TEXT        NOT NULL,
    registered_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    active           BOOLEAN     NOT NULL DEFAULT TRUE
);

-- Index for enricher lookups by device_id (already covered by PK, listed for clarity)
-- Index for listing devices by location (used in dashboard queries)
CREATE INDEX devices_location_id_idx ON devices(location_id);

-- +goose Down
DROP INDEX IF EXISTS devices_location_id_idx;
DROP TABLE IF EXISTS devices;
