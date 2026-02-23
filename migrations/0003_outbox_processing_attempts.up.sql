ALTER TABLE outbox
  ADD COLUMN IF NOT EXISTS processing_started_at TIMESTAMPTZ NULL;

ALTER TABLE outbox
  ADD COLUMN IF NOT EXISTS attempts INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS outbox_processing_started_idx
  ON outbox (status, processing_started_at);
