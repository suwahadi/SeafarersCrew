package admin

import (
	"context"
	"net/http"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
)

// FaviconSource finds the website favicon.
type FaviconSource interface {
	Favicon(ctx context.Context) (*model.Image, error)
}

// BrandingHandler tells the admin panel which favicon to show.
type BrandingHandler struct {
	source FaviconSource
	urls   resource.MediaURLer
}

// NewBrandingHandler builds a BrandingHandler.
func NewBrandingHandler(source FaviconSource, urls resource.MediaURLer) *BrandingHandler {
	return &BrandingHandler{source: source, urls: urls}
}

type brandingResponse struct {
	Favicon *resource.PublicImage `json:"favicon"`
}

// Show handles GET /branding, which needs no session so the login page can use it.
func (h *BrandingHandler) Show(w http.ResponseWriter, r *http.Request) {
	img, err := h.source.Favicon(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, brandingResponse{Favicon: resource.NewPublicImage(img, h.urls, "")})
}
