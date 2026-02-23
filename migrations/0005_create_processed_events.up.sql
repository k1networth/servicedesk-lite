CREATE TABLE IF NOT EXISTS processed_events (
  event_id      UUID PRIMARY KEY,
  event_type    TEXT NOT NULL,
  aggregate     TEXT NOT NULL,
  aggregate_id  TEXT NOT NULL,
  payload       JSONB NOT NULL,
  status        TEXT NOT NULL DEFAULT 'processing',
  attempts      INT NOT NULL DEFAULT 0,
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  processed_at  TIMESTAMPTZ NULL,
  last_error    TEXT NULL
);

CREATE INDEX IF NOT EXISTS processed_events_status_idx
  ON processed_events (status, updated_at);
