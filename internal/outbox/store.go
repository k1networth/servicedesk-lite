package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Event struct {
	ID          int64
	EventID     string
	Aggregate   string
	AggregateID string
	EventType   string
	Payload     json.RawMessage
	CreatedAt   time.Time
	Attempts    int
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) ResetStuck(ctx context.Context, processingTimeout time.Duration) (int64, error) {
	if processingTimeout <= 0 {
		processingTimeout = 30 * time.Second
	}
	const q = `
UPDATE outbox
SET status = 'pending',
    processing_started_at = NULL,
    next_retry_at = now(),
    last_error = 'processing timeout'
WHERE status = 'processing'
  AND processing_started_at IS NOT NULL
  AND processing_started_at < now() - $1::interval;
`
	res, err := s.db.ExecContext(ctx, q, fmt.Sprintf("%fs", processingTimeout.Seconds()))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) ClaimPending(ctx context.Context, batchSize int) ([]Event, error) {
	if batchSize <= 0 {
		batchSize = 50
	}

	const q = `
WITH cte AS (
  SELECT id
  FROM outbox
  WHERE status = 'pending'
    AND next_retry_at <= now()
  ORDER BY created_at
  LIMIT $1
  FOR UPDATE SKIP LOCKED
)
UPDATE outbox o
SET status = 'processing',
    processing_started_at = now(),
    attempts = attempts + 1,
    updated_at = now()
FROM cte
WHERE o.id = cte.id
RETURNING o.id, o.event_id, o.aggregate, o.aggregate_id, o.event_type, o.payload, o.created_at, o.attempts;
`

	rows, err := s.db.QueryContext(ctx, q, batchSize)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.EventID, &e.Aggregate, &e.AggregateID, &e.EventType, &e.Payload, &e.CreatedAt, &e.Attempts); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) MarkSent(ctx context.Context, id int64) error {
	const q = `
UPDATE outbox
SET status = 'sent',
    sent_at = now(),
    processing_started_at = NULL,
    last_error = NULL,
    updated_at = now()
WHERE id = $1;
`
	_, err := s.db.ExecContext(ctx, q, id)
	return err
}

func (s *Store) MarkFailed(ctx context.Context, id int64, nextRetryAt time.Time, errMsg string) error {
	const q = `
UPDATE outbox
SET status = 'pending',
    processing_started_at = NULL,
    next_retry_at = $2,
    last_error = $3,
    updated_at = now()
WHERE id = $1;
`
	_, err := s.db.ExecContext(ctx, q, id, nextRetryAt, errMsg)
	return err
}

func (s *Store) LagSeconds(ctx context.Context) (float64, error) {
	const q = `
SELECT EXTRACT(EPOCH FROM (now() - created_at))
FROM outbox
WHERE status = 'pending'
ORDER BY created_at
LIMIT 1;
`
	var v sql.NullFloat64
	if err := s.db.QueryRowContext(ctx, q).Scan(&v); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	if !v.Valid {
		return 0, nil
	}
	return v.Float64, nil
}
