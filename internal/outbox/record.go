package outbox

import (
	"encoding/json"
	"time"
)

type Record struct {
	ID                  int64
	Aggregate           string
	AggregateID         string
	EventType           string
	Payload             json.RawMessage
	CreatedAt           time.Time
	Attempts            int
	ProcessingStartedAt time.Time
}
