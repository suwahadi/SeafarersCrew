// Package apperr defines typed application errors mapped to HTTP responses.
package apperr

import (
	"errors"
	"net/http"
)

// Error is a client-facing error with a stable code and optional field errors.
type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
	cause   error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return e.Code + ": " + e.Message + ": " + e.cause.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *Error) Unwrap() error { return e.cause }

// Wrap returns a copy of e that keeps cause for logging.
func (e *Error) Wrap(cause error) *Error {
	c := *e
	c.cause = cause
	return &c
}

// New builds an Error.
func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// Validation builds a 422 error carrying per-field messages.
func Validation(fields map[string]string) *Error {
	return &Error{Status: http.StatusUnprocessableEntity, Code: "validation_failed", Message: "The given data is invalid.", Fields: fields}
}

// NotFound builds a 404 error for the named resource.
func NotFound(resource string) *Error {
	return &Error{Status: http.StatusNotFound, Code: "not_found", Message: resource + " not found."}
}

// Conflict builds a 409 error.
func Conflict(code, message string) *Error {
	return &Error{Status: http.StatusConflict, Code: code, Message: message}
}

// Common errors shared across handlers.
var (
	ErrUnauthenticated        = New(http.StatusUnauthorized, "unauthenticated", "Authentication required.")
	ErrInvalidCredentials     = New(http.StatusUnauthorized, "invalid_credentials", "These credentials do not match our records.")
	ErrTooManyAttempts        = New(http.StatusTooManyRequests, "too_many_attempts", "Too many attempts. Please try again later.")
	ErrCSRF                   = New(http.StatusForbidden, "csrf_mismatch", "Invalid or missing CSRF token.")
	ErrPasswordChangeRequired = New(http.StatusForbidden, "password_change_required", "You must change your password before continuing.")
	ErrBadRequest             = New(http.StatusBadRequest, "bad_request", "The request could not be understood.")
	ErrPayloadTooLarge        = New(http.StatusRequestEntityTooLarge, "payload_too_large", "The request body is too large.")
	ErrInternal               = New(http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
)

// As extracts an *Error from err.
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}
