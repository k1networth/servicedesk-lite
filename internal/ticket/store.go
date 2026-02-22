package ticket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
)

var ErrNotFound = errors.New("ticket not found")

type Store interface {
	Create(ctx context.Context, t Ticket) (Ticket, error)
	Get(ctx context.Context, id string) (Ticket, error)
}

type InMemoryStore struct {
	mu   sync.RWMutex
	byID map[string]Ticket
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		byID: make(map[string]Ticket),
	}
}

func (s *InMemoryStore) Create(ctx context.Context, t Ticket) (Ticket, error) {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	if t.ID == "" {
		t.ID = newID()
	}

	s.byID[t.ID] = t
	return t, nil
}

func (s *InMemoryStore) Get(ctx context.Context, id string) (Ticket, error) {
	_ = ctx

	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.byID[id]
	if !ok {
		return Ticket{}, ErrNotFound
	}
	return t, nil
}

func newID() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b[:])
}
