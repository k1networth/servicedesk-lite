package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/k1networth/servicedesk-lite/internal/ticket/model"
	"github.com/k1networth/servicedesk-lite/internal/ticket/repo"
)

type Repo struct {
	db *sql.DB
}

func New(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(ctx context.Context, t model.Ticket) (model.Ticket, error) {
	const q = `
INSERT INTO tickets (id, title, description, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, title, description, status, created_at, updated_at;
`
	var out model.Ticket
	err := r.db.QueryRowContext(
		ctx, q,
		t.ID, t.Title, t.Description, t.Status, t.CreatedAt, t.UpdatedAt,
	).Scan(&out.ID, &out.Title, &out.Description, &out.Status, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return model.Ticket{}, err
	}
	return out, nil
}

func (r *Repo) GetByID(ctx context.Context, id string) (model.Ticket, error) {
	const q = `
SELECT id, title, description, status, created_at, updated_at
FROM tickets
WHERE id = $1;
`
	var out model.Ticket
	err := r.db.QueryRowContext(ctx, q, id).
		Scan(&out.ID, &out.Title, &out.Description, &out.Status, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Ticket{}, repo.ErrNotFound
		}
		return model.Ticket{}, err
	}
	return out, nil
}
