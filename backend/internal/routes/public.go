package routes

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler"
	"github.com/suwahadi/seafarerscrew/backend/internal/handler/public"
	"github.com/suwahadi/seafarerscrew/backend/internal/middleware"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// PublicPrefix is the base path of the public API.
const PublicPrefix = "/api/public/v1"

// PublicCacheTTL is how long a public read may be served from Redis.
const PublicCacheTTL = 60 * time.Second

// PublicDeps carries the controllers and middleware of the public API.
type PublicDeps struct {
	Origin    string
	Counter   middleware.HitCounter
	Cache     middleware.PublicCacheStore
	Media     *public.MediaHandler
	Redirects *public.RedirectHandler
	Career    PublicCareerRoutes
}

// Public mounts the public API.
func Public(r chi.Router, d PublicDeps) {
	r.Route(PublicPrefix, func(r chi.Router) {
		r.Use(middleware.CORS(d.Origin))
		r.Get("/media/*", d.Media.Show)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RateLimit(d.Counter, "public-api", 600, time.Minute))
			r.With(cached(d, model.TagRedirects)).Get("/redirects/*", d.Redirects.Show)
			PublicCareer(r, d)
		})
	})
}

func cached(d PublicDeps, tags ...string) func(http.Handler) http.Handler {
	return middleware.PublicCache(d.Cache, PublicCacheTTL, tags...)
}

// Health mounts liveness and readiness probes.
func Health(r chi.Router, h *handler.HealthHandler) {
	r.Get("/healthz", h.Live)
	r.Get("/readyz", h.Ready)
}
