package public

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type fakeMedia map[string][]byte

func (f fakeMedia) Object(_ context.Context, p string) ([]byte, string, error) {
	if v, ok := f[p]; ok {
		return v, "image/svg+xml", nil
	}
	return nil, "", apperr.NotFound("File")
}

func mediaRouter(m fakeMedia) http.Handler {
	r := chi.NewRouter()
	r.Get("/media/*", NewMediaHandler(m).Show)
	return r
}

func TestMediaServesImmutableSandboxedFiles(t *testing.T) {
	rec := httptest.NewRecorder()
	mediaRouter(fakeMedia{"flag/2026/10/id.svg": []byte("<svg/>")}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/media/flag/2026/10/id.svg", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "<svg/>", rec.Body.String())
	assert.Equal(t, "image/svg+xml", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Cache-Control"), "immutable")
	assert.Contains(t, rec.Header().Get("Content-Security-Policy"), "sandbox")
}

func TestMediaReturnsNotFoundForUnknownPaths(t *testing.T) {
	rec := httptest.NewRecorder()
	mediaRouter(fakeMedia{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/media/x.webp", nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
