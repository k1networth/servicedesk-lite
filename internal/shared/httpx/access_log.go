package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/k1networth/servicedesk-lite/internal/shared/requestid"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func AccessLog(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			sw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			dur := time.Since(start)
			rid := requestid.Get(r.Context())

			log.Info("http_request",
				slog.String("request_id", rid),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sw.status),
				slog.Int64("duration_ms", dur.Milliseconds()),
			)
		})
	}
}
