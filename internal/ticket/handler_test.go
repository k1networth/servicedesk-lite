package ticket_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/k1networth/servicedesk-lite/internal/shared/httpx"
	"github.com/k1networth/servicedesk-lite/internal/ticket"
)

func testLogger() *slog.Logger {
	h := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(h).With(
		slog.String("app", "test"),
		slog.String("env", "test"),
	)
}

func newTestServer() *httptest.Server {
	log := testLogger()

	store := ticket.NewInMemoryStore()
	ticketH := &ticket.Handler{Log: log, Store: store}

	handler := httpx.NewRouter(log, ticketH)
	return httptest.NewServer(handler)
}

func TestCreateTicket201(t *testing.T) {
	srv := newTestServer()
	t.Cleanup(srv.Close)

	body := []byte(`{"title":"Printer is broken","description":"Office 3rd floor"}`)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/tickets", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected %d, got %d, body=%s", http.StatusCreated, resp.StatusCode, string(b))
	}

	var got ticket.Ticket
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID == "" {
		t.Fatalf("expected id to be set")
	}
	if got.Status != "open" {
		t.Fatalf("expected status %q, got %q", "open", got.Status)
	}
	if got.Title != "Printer is broken" {
		t.Fatalf("expected title %q, got %q", "Printer is broken", got.Title)
	}

	if got.CreatedAt.IsZero() {
		t.Fatalf("expected created_at to be set")
	}

	if rid := resp.Header.Get("X-Request-Id"); rid == "" {
		t.Fatalf("expected X-Request-Id header to be set")
	}
}

func TestCreateTicketValidation400(t *testing.T) {
	srv := newTestServer()
	t.Cleanup(srv.Close)

	body := []byte(`{"title":""}`)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/tickets", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected %d, got %d, body=%s", http.StatusBadRequest, resp.StatusCode, string(b))
	}

	var er struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if er.Error.Code != "validation_error" {
		t.Fatalf("expected code %q, got %q", "validation_error", er.Error.Code)
	}
	if er.Error.Message == "" {
		t.Fatalf("expected message to be set")
	}
}

func TestGetTicket200AfterCreate(t *testing.T) {
	srv := newTestServer()
	t.Cleanup(srv.Close)

	createBody := []byte(`{"title":"Network down","description":"No internet"}`)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/tickets", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	createResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = createResp.Body.Close() }()

	if createResp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(createResp.Body)
		t.Fatalf("expected %d, got %d, body=%s", http.StatusCreated, createResp.StatusCode, string(b))
	}

	var created ticket.Ticket
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected id to be set")
	}

	getResp, err := http.Get(srv.URL + "/tickets/" + created.ID)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	defer func() { _ = getResp.Body.Close() }()

	if getResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(getResp.Body)
		t.Fatalf("expected %d, got %d, body=%s", http.StatusOK, getResp.StatusCode, string(b))
	}

	var got ticket.Ticket
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode get response: %v", err)
	}

	if got.ID != created.ID {
		t.Fatalf("expected id %q, got %q", created.ID, got.ID)
	}
	if got.Title != "Network down" {
		t.Fatalf("expected title %q, got %q", "Network down", got.Title)
	}
}
