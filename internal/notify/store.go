package notify

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type ProcessedEvent struct {
	EventID     string
	EventType   string
	Aggregate   string
	AggregateID string
	Payload     json.RawMessage
	Status      string
	Attempts    int
	LastError   sql.NullString
	ProcessedAt sql.NullTime
	UpdatedAt   time.Time
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// StartProcessing ensures we have a row and marks it as processing.
// Returns (shouldProcess=false) when the event is already done/failed.
// attempts is the (post-increment) attempts counter for this event.
func (s *Store) StartProcessing(ctx context.Context, e ProcessedEvent) (shouldProcess bool, attempts int, status string, err error) {
	const q = `
INSERT INTO processed_events (event_id, event_type, aggregate, aggregate_id, payload, status, attempts, updated_at)
VALUES ($1,$2,$3,$4,$5,'processing',1,now())
ON CONFLICT (event_id) DO UPDATE
SET attempts = CASE
        WHEN processed_events.status IN ('done','failed') THEN processed_events.attempts
        ELSE processed_events.attempts + 1
    END,
    status = CASE
        WHEN processed_events.status IN ('done','failed') THEN processed_events.status
        ELSE 'processing'
    END,
    updated_at = now()
RETURNING status, attempts;
`
	err = s.db.QueryRowContext(ctx, q, e.EventID, e.EventType, e.Aggregate, e.AggregateID, e.Payload).Scan(&status, &attempts)
	if err != nil {
		return false, 0, "", err
	}
	if status == "done" || status == "failed" {
		return false, attempts, status, nil
	}
	return true, attempts, status, nil
}

func (s *Store) MarkDone(ctx context.Context, eventID string) error {
	const q = `
UPDATE processed_events
SET status='done', processed_at=now(), last_error=NULL, updated_at=now()
WHERE event_id=$1;
`
	_, err := s.db.ExecContext(ctx, q, eventID)
	return err
}

func (s *Store) MarkFailed(ctx context.Context, eventID string, errMsg string) error {
	const q = `
UPDATE processed_events
SET status='processing', last_error=$2, updated_at=now()
WHERE event_id=$1;
`
	_, err := s.db.ExecContext(ctx, q, eventID, errMsg)
	return err
}

func (s *Store) MarkDead(ctx context.Context, eventID string, errMsg string) error {
	const q = `
UPDATE processed_events
SET status='failed', failed_at = COALESCE(failed_at, now()), last_error=$2, updated_at=now()
WHERE event_id=$1;
`
	_, err := s.db.ExecContext(ctx, q, eventID, errMsg)
	return err
}
