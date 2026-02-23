package events

import (
	"encoding/json"
	"time"
)

type Envelope struct {
	EventID     string          `json:"event_id"`
	EventType   string          `json:"event_type"`
	OccurredAt  time.Time       `json:"occurred_at"`
	Aggregate   string          `json:"aggregate"`
	AggregateID string          `json:"aggregate_id"`
	RequestID   string          `json:"request_id,omitempty"`
	Payload     json.RawMessage `json:"payload"`
}
