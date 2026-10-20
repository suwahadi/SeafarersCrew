// Package logctx carries a request-scoped slog.Logger in a context.
package logctx

import (
	"context"
	"log/slog"
)

type key struct{}

// With returns ctx carrying logger.
func With(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, key{}, logger)
}

// From returns the logger in ctx or slog.Default.
func From(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(key{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
