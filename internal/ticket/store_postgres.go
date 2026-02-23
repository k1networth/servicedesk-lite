package ticket

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/k1networth/servicedesk-lite/internal/shared/requestid"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Create(ctx context.Context, t Ticket) (Ticket, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return Ticket{}, err
	}
	defer func() { _ = tx.Rollback() }()

	const qTicket = `
INSERT INTO tickets (id, title, description, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, title, description, status, created_at, updated_at;
`
	var out Ticket
	err = tx.QueryRowContext(ctx, qTicket,
		t.ID, t.Title, t.Description, t.Status, t.CreatedAt, t.UpdatedAt,
	).Scan(&out.ID, &out.Title, &out.Description, &out.Status, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return Ticket{}, err
	}

	rid := requestid.Get(ctx)
	payloadObj := map[string]any{
		"ticket_id":  out.ID,
		"title":      out.Title,
		"status":     out.Status,
		"created_at": out.CreatedAt,
		"request_id": rid,
	}
	payload, err := json.Marshal(payloadObj)
	if err != nil {
		return Ticket{}, err
	}

	const qOutbox = `
INSERT INTO outbox (aggregate, aggregate_id, event_type, payload)
VALUES ($1, $2, $3, $4::jsonb);
`
	_, err = tx.ExecContext(ctx, qOutbox,
		"ticket", out.ID, "ticket.created", payload,
	)
	if err != nil {
		return Ticket{}, err
	}

	if err := tx.Commit(); err != nil {
		return Ticket{}, err
	}

	return out, nil
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Ticket, error) {
	const q = `
SELECT id, title, description, status, created_at, updated_at
FROM tickets
WHERE id = $1;
`
	var out Ticket
	err := s.db.QueryRowContext(ctx, q, id).
		Scan(&out.ID, &out.Title, &out.Description, &out.Status, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Ticket{}, ErrNotFound
		}
		return Ticket{}, err
	}
	return out, nil
}
