package ticket

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	Log   *slog.Logger
	Store Store
}

func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req CreateTicketRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		msg := "invalid json"
		if errors.Is(err, io.EOF) {
			msg = "empty body"
		}
		WriteError(w, http.StatusBadRequest, "validation_error", msg)
		return
	}

	if dec.More() {
		WriteError(w, http.StatusBadRequest, "validation_error", "invalid json")
		return
	}

	if err := req.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	t := Ticket{
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Status:      "open",
		CreatedAt:   time.Now().UTC(),
	}

	created, err := h.Store.Create(r.Context(), t)
	if err != nil {
		h.Log.Error("ticket_create_failed", slog.String("err", err.Error()))
		WriteError(w, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) GetTicket(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	t, err := h.Store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		h.Log.Error("ticket_get_failed", slog.String("err", err.Error()))
		WriteError(w, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
