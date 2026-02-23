package ticket

import (
	"strings"
	"time"
)

type Ticket struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateTicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (r CreateTicketRequest) Validate() error {
	title := strings.TrimSpace(r.Title)
	if title == "" {
		return ValidationError("title is required")
	}
	if len(title) < 3 {
		return ValidationError("title must be at least 3 characters")
	}
	if len(title) > 200 {
		return ValidationError("title must be at most 200 characters")
	}

	desc := strings.TrimSpace(r.Description)
	if len(desc) > 5000 {
		return ValidationError("description must be at most 5000 characters")
	}

	return nil
}
