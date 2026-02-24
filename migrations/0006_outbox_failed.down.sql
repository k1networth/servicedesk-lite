DROP INDEX IF EXISTS outbox_failed_created_idx;

ALTER TABLE outbox
  DROP COLUMN IF EXISTS failed_at;
