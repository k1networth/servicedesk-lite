package ticket

import (
	"encoding/json"
	"net/http"
)

type apiErrorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationError string

func (e ValidationError) Error() string { return string(e) }

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(apiErrorResponse{
		Error: apiError{
			Code:    code,
			Message: message,
		},
	})
}
