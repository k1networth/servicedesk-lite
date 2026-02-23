package ticket

import (
	"context"
	"database/sql"
	"errors"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Create(ctx context.Context, t Ticket) (Ticket, error) {
	const q = `
INSERT INTO tickets (id, title, description, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, title, description, status, created_at, updated_at;
`
	var out Ticket
	err := s.db.QueryRowContext(ctx, q,
		t.ID, t.Title, t.Description, t.Status, t.CreatedAt, t.UpdatedAt,
	).Scan(&out.ID, &out.Title, &out.Description, &out.Status, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
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
