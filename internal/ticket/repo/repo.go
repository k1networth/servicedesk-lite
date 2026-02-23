package repo

import (
	"context"
	"errors"

	"github.com/k1networth/servicedesk-lite/internal/ticket/model"
)

var ErrNotFound = errors.New("not found")

type Repository interface {
	Create(ctx context.Context, t model.Ticket) (model.Ticket, error)
	GetByID(ctx context.Context, id string) (model.Ticket, error)
}
