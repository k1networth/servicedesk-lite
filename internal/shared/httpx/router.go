package httpx

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/k1networth/servicedesk-lite/internal/ticket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(log *slog.Logger, ticketH *ticket.Handler) http.Handler {
	mux := http.NewServeMux()

	reg := prometheus.NewRegistry()
	met := NewMetrics(reg)

	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	mux.Handle("/healthz", WithRoute("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})))

	mux.Handle("/readyz", WithRoute("/readyz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})))

	mux.Handle("/tickets", WithRoute("/tickets", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ticketH.CreateTicket(w, r)
	})))

	mux.Handle("/tickets/", WithRoute("/tickets/:id", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/tickets/")
		if id == "" || strings.Contains(id, "/") {
			ticket.WriteErrorR(w, r, http.StatusNotFound, "not_found", "not found")
			return
		}
		ticketH.GetTicket(w, r, id)
	})))

	var h http.Handler = mux
	h = met.Middleware(h)
	h = AccessLog(log)(h)
	h = RequestID(h)

	return h
}
