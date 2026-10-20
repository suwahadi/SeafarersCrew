// Package resource shapes API responses.
package resource

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/logctx"
)

// Meta describes a page of a list response.
type Meta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

type envelope struct {
	Data any   `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// JSON writes {"data": data} with status.
func JSON(w http.ResponseWriter, status int, data any) {
	write(w, status, envelope{Data: data})
}

// List writes {"data": data, "meta": meta}.
func List(w http.ResponseWriter, data any, meta Meta) {
	write(w, http.StatusOK, envelope{Data: data, Meta: &meta})
}

// NoContent writes 204.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error writes the error envelope; unexpected errors are logged and hidden behind a generic 500.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	e, ok := apperr.As(err)
	if !ok {
		if errors.Is(err, r.Context().Err()) && r.Context().Err() != nil {
			return
		}
		logctx.From(r.Context()).ErrorContext(r.Context(), "request failed", "error", err)
		e = apperr.ErrInternal
	}
	if e.Status >= http.StatusInternalServerError && ok {
		logctx.From(r.Context()).ErrorContext(r.Context(), "request failed", "error", err)
	}
	write(w, e.Status, map[string]errorBody{"error": {Code: e.Code, Message: e.Message, Fields: e.Fields}})
}

func write(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
