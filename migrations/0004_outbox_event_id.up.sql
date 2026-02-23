CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE outbox
  ADD COLUMN IF NOT EXISTS event_id UUID NOT NULL DEFAULT gen_random_uuid();

CREATE UNIQUE INDEX IF NOT EXISTS outbox_event_id_uq ON outbox (event_id);
