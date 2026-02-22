package ticket

import (
	"encoding/json"
	"net/http"

	"github.com/k1networth/servicedesk-lite/internal/shared/requestid"
)

type apiErrorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

type ValidationError string

func (e ValidationError) Error() string { return string(e) }

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiErrorResponse{
		Error: apiError{Code: code, Message: message},
	})
}

func WriteErrorR(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	rid := requestid.Get(r.Context())

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiErrorResponse{
		Error: apiError{Code: code, Message: message, RequestID: rid},
	})
}
