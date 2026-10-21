// Package public holds the controllers of the public API.
package public

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
)

// MediaService reads public files.
type MediaService interface {
	Object(ctx context.Context, path string) ([]byte, string, error)
}

// MediaHandler serves CMS images, logos and flags.
type MediaHandler struct {
	media MediaService
}

// NewMediaHandler builds a MediaHandler.
func NewMediaHandler(media MediaService) *MediaHandler {
	return &MediaHandler{media: media}
}

// Show handles GET /media/*.
func (h *MediaHandler) Show(w http.ResponseWriter, r *http.Request) {
	data, contentType, err := h.media.Object(r.Context(), chi.URLParam(r, "*"))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	hdr := w.Header()
	hdr.Set("Content-Type", contentType)
	hdr.Set("Content-Length", strconv.Itoa(len(data)))
	hdr.Set("Cache-Control", "public, max-age=31536000, immutable")
	hdr.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	hdr.Set("Cross-Origin-Resource-Policy", "cross-origin")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
