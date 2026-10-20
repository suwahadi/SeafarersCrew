package routes

import (
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler"
	"github.com/suwahadi/seafarerscrew/backend/internal/middleware"
)

// PublicPrefix is the base path of the public API.
const PublicPrefix = "/api/public/v1"

// PublicDeps carries the controllers and middleware of the public API.
type PublicDeps struct {
	Origin  string
	Counter middleware.HitCounter
}

// Public mounts the public API.
func Public(r chi.Router, d PublicDeps) {
	r.Route(PublicPrefix, func(r chi.Router) {
		r.Use(middleware.CORS(d.Origin))
		r.Use(middleware.RateLimit(d.Counter, "public-api", 600, time.Minute))
	})
}

// Health mounts liveness and readiness probes.
func Health(r chi.Router, h *handler.HealthHandler) {
	r.Get("/healthz", h.Live)
	r.Get("/readyz", h.Ready)
}
