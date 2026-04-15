-- +goose Up
-- Dead Letter Queue table: persists Kafka messages that could not be processed.
-- A message lands here after 3 failed processing attempts. The original Kafka
-- payload is stored as BYTEA alongside metadata for debugging and replay.
--
-- Replay workflow:
--   1. Query rows WHERE reprocessed_at IS NULL
--   2. Re-publish the payload to the original_topic
--   3. UPDATE reprocessed_at = NOW() to mark as replayed
--
-- This table is write-mostly. Keep it lean — no FK constraints since the
-- device may not even be in the registry (that could be why it failed).
CREATE TABLE dead_letter_queue (
    id                 BIGSERIAL   PRIMARY KEY,
    original_topic     TEXT        NOT NULL,
    original_partition INT         NOT NULL,
    original_offset    BIGINT      NOT NULL,
    payload            BYTEA       NOT NULL,  -- original Avro wire-format bytes
    error_reason       TEXT        NOT NULL,  -- human-readable failure description
    failed_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reprocessed_at     TIMESTAMPTZ           -- NULL = not yet replayed
);

-- Index for the replay workflow query
CREATE INDEX dlq_reprocessed_at_idx ON dead_letter_queue(reprocessed_at)
    WHERE reprocessed_at IS NULL;

-- Index for inspecting failures by topic
CREATE INDEX dlq_original_topic_idx ON dead_letter_queue(original_topic, failed_at DESC);

-- +goose Down
DROP INDEX IF EXISTS dlq_original_topic_idx;
DROP INDEX IF EXISTS dlq_reprocessed_at_idx;
DROP TABLE IF EXISTS dead_letter_queue;
