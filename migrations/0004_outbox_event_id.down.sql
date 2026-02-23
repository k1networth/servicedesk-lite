DROP INDEX IF EXISTS outbox_event_id_uq;
ALTER TABLE outbox DROP COLUMN IF EXISTS event_id;
