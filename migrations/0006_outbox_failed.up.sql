ALTER TABLE outbox
  ADD COLUMN IF NOT EXISTS failed_at TIMESTAMPTZ NULL;

-- speed up querying failed events in dashboards/ops
CREATE INDEX IF NOT EXISTS outbox_failed_created_idx
  ON outbox (status, created_at)
  WHERE status = 'failed';
