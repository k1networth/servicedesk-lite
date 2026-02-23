DROP INDEX IF EXISTS outbox_pending_ready_idx;

ALTER TABLE outbox
  DROP COLUMN IF EXISTS processing_started_at,
  DROP COLUMN IF EXISTS attempts,
  DROP COLUMN IF EXISTS next_retry_at,
  DROP COLUMN IF EXISTS last_error,
  DROP COLUMN IF EXISTS updated_at;
