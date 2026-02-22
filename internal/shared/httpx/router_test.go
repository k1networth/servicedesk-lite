package httpx_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/k1networth/servicedesk-lite/internal/shared/httpx"
)

func testLogger() *slog.Logger {
	h := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(h).With(
		slog.String("app", "test"),
		slog.String("env", "test"),
	)
}

func TestHealthzReturns200AndBodyOK(t *testing.T) {
	log := testLogger()
	h := httpx.NewRouter(log)

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", string(body))
	}
}

func TestRequestIDGeneratedIfMissing(t *testing.T) {
	log := testLogger()
	h := httpx.NewRouter(log)

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/healthz", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	got := resp.Header.Get("X-Request-Id")
	if got == "" {
		t.Fatalf("expected X-Request-Id header to be set")
	}

	re := regexp.MustCompile(`^[0-9a-f]{32}$`)
	if !re.MatchString(got) {
		t.Fatalf("expected generated request id to look like 32-char hex, got %q", got)
	}
}

func TestRequestIDPreservedIfProvided(t *testing.T) {
	log := testLogger()
	h := httpx.NewRouter(log)

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/healthz", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	req.Header.Set("X-Request-Id", "test123")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	got := resp.Header.Get("X-Request-Id")
	if got != "test123" {
		t.Fatalf("expected X-Request-Id %q, got %q", "test123", got)
	}
}
