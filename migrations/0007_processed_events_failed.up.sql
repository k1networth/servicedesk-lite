ALTER TABLE processed_events
  ADD COLUMN IF NOT EXISTS failed_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS processed_events_failed_idx
  ON processed_events (status, updated_at)
  WHERE status = 'failed';
