DROP INDEX IF EXISTS outbox_processing_started_idx;

ALTER TABLE outbox
  DROP COLUMN IF EXISTS processing_started_at;

ALTER TABLE outbox
  DROP COLUMN IF EXISTS attempts;
