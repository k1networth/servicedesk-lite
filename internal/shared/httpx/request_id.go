package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/k1networth/servicedesk-lite/internal/shared/requestid"
)

const requestIDHeader = "X-Request-Id"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := strings.TrimSpace(r.Header.Get(requestIDHeader))
		if rid == "" {
			rid = newRequestID()
		}

		w.Header().Set(requestIDHeader, rid)

		ctx := requestid.With(r.Context(), rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(b[:])
}
