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
// Returns (shouldProcess=false) when the event is already done.
func (s *Store) StartProcessing(ctx context.Context, e ProcessedEvent) (bool, error) {
	// Upsert: if already done -> keep done and return false.
	// Otherwise bump attempts and keep processing.
	const q = `
INSERT INTO processed_events (event_id, event_type, aggregate, aggregate_id, payload, status, attempts, updated_at)
VALUES ($1,$2,$3,$4,$5,'processing',1,now())
ON CONFLICT (event_id) DO UPDATE
SET attempts = processed_events.attempts + 1,
    updated_at = now()
RETURNING status;
`
	var status string
	err := s.db.QueryRowContext(ctx, q, e.EventID, e.EventType, e.Aggregate, e.AggregateID, e.Payload).Scan(&status)
	if err != nil {
		return false, err
	}
	if status == "done" {
		return false, nil
	}
	return true, nil
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
