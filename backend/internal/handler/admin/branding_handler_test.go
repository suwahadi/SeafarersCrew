package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

type stubFavicon struct {
	img *model.Image
	err error
}

func (s stubFavicon) Favicon(context.Context) (*model.Image, error) { return s.img, s.err }

type prefixURLs struct{}

func (prefixURLs) URL(key string) string {
	rest, ok := strings.CutPrefix(key, "public/")
	if !ok {
		return ""
	}
	return "https://api.example.com/media/" + rest
}

func TestBrandingReturnsTheFaviconUrl(t *testing.T) {
	h := NewBrandingHandler(stubFavicon{img: &model.Image{StorageKey: "public/logo/2026/10/fav.webp", MimeType: "image/webp", Width: 64, Height: 64}}, prefixURLs{})
	rec := httptest.NewRecorder()
	h.Show(rec, httptest.NewRequest(http.MethodGet, "/branding", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"url":"https://api.example.com/media/logo/2026/10/fav.webp"`)
}

func TestBrandingReturnsNullWhenNoFaviconIsSet(t *testing.T) {
	rec := httptest.NewRecorder()
	NewBrandingHandler(stubFavicon{}, prefixURLs{}).Show(rec, httptest.NewRequest(http.MethodGet, "/branding", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"data":{"favicon":null}}`, rec.Body.String())
}

func TestBrandingReportsLookupFailures(t *testing.T) {
	rec := httptest.NewRecorder()
	NewBrandingHandler(stubFavicon{err: errors.New("db down")}, prefixURLs{}).Show(rec, httptest.NewRequest(http.MethodGet, "/branding", nil))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
