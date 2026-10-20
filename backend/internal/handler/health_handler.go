// Package handler holds controllers shared by both APIs.
package handler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/logctx"
)

// Pinger checks a dependency.
type Pinger interface {
	Ping(ctx context.Context) error
}

// PingFunc adapts a function to Pinger.
type PingFunc func(ctx context.Context) error

// Ping implements Pinger.
func (f PingFunc) Ping(ctx context.Context) error { return f(ctx) }

// HealthHandler serves liveness and readiness probes.
type HealthHandler struct {
	deps map[string]Pinger
}

// NewHealthHandler returns a HealthHandler checking deps on readiness.
func NewHealthHandler(deps map[string]Pinger) *HealthHandler {
	return &HealthHandler{deps: deps}
}

// Live handles GET /healthz.
func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
	resource.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready handles GET /readyz.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	var mu sync.Mutex
	results := make(map[string]string, len(h.deps))
	status := http.StatusOK
	var g errgroup.Group
	for name, dep := range h.deps {
		g.Go(func() error {
			err := dep.Ping(ctx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results[name] = "unavailable"
				status = http.StatusServiceUnavailable
				logctx.From(ctx).WarnContext(ctx, "readiness check failed", "dependency", name, "error", err)
				return nil
			}
			results[name] = "ok"
			return nil
		})
	}
	_ = g.Wait()
	resource.JSON(w, status, results)
}
