package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

const requestIDHeader = "X-Request-Id"

type ctxKeyRequestID struct{}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := strings.TrimSpace(r.Header.Get(requestIDHeader))
		if rid == "" {
			rid = newRequestID()
		}

		w.Header().Set(requestIDHeader, rid)

		ctx := context.WithValue(r.Context(), ctxKeyRequestID{}, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetRequestID(ctx context.Context) string {
	v := ctx.Value(ctxKeyRequestID{})
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func newRequestID() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(b[:])
}
