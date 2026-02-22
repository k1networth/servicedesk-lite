package httpx

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/k1networth/servicedesk-lite/internal/ticket"
)

func NewRouter(log *slog.Logger, ticketH *ticket.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# metrics placeholder\n"))
	})

	mux.HandleFunc("/tickets", func(w http.ResponseWriter, r *http.Request) {
		ticketH.CreateTicket(w, r)
	})

	mux.HandleFunc("/tickets/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/tickets/")
		if id == "" || strings.Contains(id, "/") {
			ticket.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		ticketH.GetTicket(w, r, id)
	})

	var h http.Handler = mux
	h = RequestID(h)
	h = AccessLog(log)(h)

	return h
}
