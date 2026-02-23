CREATE TABLE IF NOT EXISTS outbox (
  id           BIGSERIAL PRIMARY KEY,
  aggregate    TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  event_type   TEXT NOT NULL,
  payload      JSONB NOT NULL,
  status       TEXT NOT NULL DEFAULT 'pending',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  sent_at      TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS outbox_status_created_idx
  ON outbox (status, created_at);

CREATE INDEX IF NOT EXISTS outbox_aggregate_idx
  ON outbox (aggregate, aggregate_id);