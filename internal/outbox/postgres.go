package outbox

import (
	"context"
	"database/sql"
	"time"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) ClaimPending(ctx context.Context, limit int) ([]Record, error) {
	const q = `
WITH cte AS (
  SELECT id
  FROM outbox
  WHERE status = 'pending'
  ORDER BY created_at
  FOR UPDATE SKIP LOCKED
  LIMIT $1
)
UPDATE outbox o
SET status = 'processing',
    processing_started_at = now(),
    attempts = o.attempts + 1
FROM cte
WHERE o.id = cte.id
RETURNING o.id, o.aggregate, o.aggregate_id, o.event_type, o.payload,
          o.created_at, o.attempts, o.processing_started_at;
`

	rows, err := r.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []Record
	for rows.Next() {
		var rec Record
		var payload []byte
		if err := rows.Scan(
			&rec.ID,
			&rec.Aggregate,
			&rec.AggregateID,
			&rec.EventType,
			&payload,
			&rec.CreatedAt,
			&rec.Attempts,
			&rec.ProcessingStartedAt,
		); err != nil {
			return nil, err
		}
		rec.Payload = payload
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PostgresRepo) MarkSent(ctx context.Context, id int64) error {
	const q = `
UPDATE outbox
SET status = 'sent', sent_at = now(), processing_started_at = NULL
WHERE id = $1;
`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *PostgresRepo) MarkPending(ctx context.Context, id int64) error {
	const q = `
UPDATE outbox
SET status = 'pending', processing_started_at = NULL
WHERE id = $1;
`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *PostgresRepo) RequeueStuck(ctx context.Context, timeout time.Duration) (int64, error) {
	threshold := time.Now().UTC().Add(-timeout)
	const q = `
UPDATE outbox
SET status = 'pending', processing_started_at = NULL
WHERE status = 'processing' AND processing_started_at < $1;
`
	res, err := r.db.ExecContext(ctx, q, threshold)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}
