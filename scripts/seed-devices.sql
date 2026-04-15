-- seed-devices.sql — Seeds 1000 test devices into the device registry.
--
-- Usage:
--   psql "$POSTGRES_DSN" -f scripts/seed-devices.sql
--
-- Device distribution:
--   - 1000 devices spread across 10 logical locations (100 per location)
--   - 3 firmware versions in rotation (v1.0.0, v1.1.0, v1.2.0)
--   - 5% of devices marked inactive (models real-world device churn)
--   - registered_at spread randomly over the last year
--
-- Idempotent: uses INSERT ... ON CONFLICT DO NOTHING so re-running is safe.

INSERT INTO devices (device_id, name, location_id, firmware_version, registered_at, active)
SELECT
    'device-' || LPAD(gs::TEXT, 4, '0')        AS device_id,
    'Sensor-' || LPAD(gs::TEXT, 4, '0')         AS name,
    'location-' || LPAD(((gs % 10) + 1)::TEXT, 2, '0') AS location_id,
    CASE (gs % 3)
        WHEN 0 THEN 'v1.0.0'
        WHEN 1 THEN 'v1.1.0'
        ELSE        'v1.2.0'
    END                                          AS firmware_version,
    NOW() - (RANDOM() * INTERVAL '365 days')    AS registered_at,
    (gs % 20 != 0)                              AS active   -- 5% inactive
FROM generate_series(1, 1000) AS gs
ON CONFLICT (device_id) DO NOTHING;

-- Summary of what was seeded
SELECT
    location_id,
    COUNT(*)                                    AS total_devices,
    COUNT(*) FILTER (WHERE active)              AS active_devices,
    COUNT(*) FILTER (WHERE NOT active)          AS inactive_devices,
    COUNT(DISTINCT firmware_version)            AS firmware_versions
FROM devices
GROUP BY location_id
ORDER BY location_id;
