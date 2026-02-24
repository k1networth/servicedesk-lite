DROP INDEX IF EXISTS processed_events_failed_idx;

ALTER TABLE processed_events
  DROP COLUMN IF EXISTS failed_at;
