// Package middleware holds HTTP middleware shared by the admin and public APIs.
package middleware

import (
	"context"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

type ctxKey int

const (
	adminKey ctxKey = iota
	sessionKey
	requestIDKey
	clientIPKey
)

// Admin returns the authenticated admin from ctx.
func Admin(ctx context.Context) (model.Admin, bool) {
	a, ok := ctx.Value(adminKey).(model.Admin)
	return a, ok
}

// SessionID returns the raw session ID from ctx.
func SessionID(ctx context.Context) string {
	s, _ := ctx.Value(sessionKey).(string)
	return s
}

// RequestIDFrom returns the request ID from ctx.
func RequestIDFrom(ctx context.Context) string {
	s, _ := ctx.Value(requestIDKey).(string)
	return s
}

// ClientIP returns the resolved client IP from ctx.
func ClientIP(ctx context.Context) string {
	s, _ := ctx.Value(clientIPKey).(string)
	return s
}

// WithAdmin returns ctx carrying the admin and session ID.
func WithAdmin(ctx context.Context, a model.Admin, sessionID string) context.Context {
	return context.WithValue(context.WithValue(ctx, adminKey, a), sessionKey, sessionID)
}
