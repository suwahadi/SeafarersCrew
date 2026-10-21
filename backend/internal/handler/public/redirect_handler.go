package public

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
)

// RedirectService resolves legacy paths.
type RedirectService interface {
	Resolve(ctx context.Context, path string) (model.Redirect, error)
}

// RedirectHandler serves slug redirects to the website.
type RedirectHandler struct {
	redirects RedirectService
}

// NewRedirectHandler builds a RedirectHandler.
func NewRedirectHandler(redirects RedirectService) *RedirectHandler {
	return &RedirectHandler{redirects: redirects}
}

// Show handles GET /redirects/*.
func (h *RedirectHandler) Show(w http.ResponseWriter, r *http.Request) {
	rd, err := h.redirects.Resolve(r.Context(), chi.URLParam(r, "*"))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, rd)
}
