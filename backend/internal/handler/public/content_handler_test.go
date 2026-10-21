package public

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type stubContent struct {
	slug          string
	page, perPage int
	missingPage   bool
	articles      []model.News
}

type noURLs struct{}

func (noURLs) URL(string) string { return "" }

func (s *stubContent) Site(context.Context) (model.SiteData, error) {
	return model.SiteData{Settings: map[string]map[string]any{"general": {"site_name": "Seafarers Crew"}}}, nil
}

func (s *stubContent) Home(context.Context) (model.HomeData, error) {
	return model.HomeData{Sections: []string{"hero"}}, nil
}

func (s *stubContent) Page(_ context.Context, slug string) (model.PageData, error) {
	s.slug = slug
	if s.missingPage {
		return model.PageData{}, apperr.NotFound("Page")
	}
	return model.PageData{Page: model.Page{Title: "About", Slug: slug}}, nil
}

func (s *stubContent) NewsList(_ context.Context, page, perPage int) (model.NewsPageData, error) {
	s.page, s.perPage = page, perPage
	return model.NewsPageData{Articles: s.articles, Page: 2, PerPage: 9, Total: 20}, nil
}

func (s *stubContent) NewsDetail(_ context.Context, slug string) (model.NewsDetailData, error) {
	if slug != "crew-change" {
		return model.NewsDetailData{}, apperr.NotFound("Article")
	}
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	return model.NewsDetailData{Article: model.News{Title: "Crew change", Slug: slug, PublishedAt: &at}}, nil
}

func (s *stubContent) Gallery(context.Context) ([]model.GalleryImage, error) { return nil, nil }

func (s *stubContent) Sitemap(context.Context) (model.SitemapData, error) {
	return model.SitemapData{}, nil
}

func contentRouter(svc *stubContent) http.Handler {
	h := NewContentHandler(svc, noURLs{})
	r := chi.NewRouter()
	r.Get("/site", h.Site)
	r.Get("/home", h.Home)
	r.Get("/pages/{slug}", h.Page)
	r.Get("/news", h.NewsIndex)
	r.Get("/news/{slug}", h.NewsShow)
	r.Get("/gallery", h.Gallery)
	r.Get("/sitemap-data", h.Sitemap)
	return r
}

func get(h http.Handler, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestContentEndpointsWrapDataInTheEnvelope(t *testing.T) {
	r := contentRouter(&stubContent{})
	for _, target := range []string{"/site", "/home", "/pages/about", "/news/crew-change", "/gallery", "/sitemap-data"} {
		rec := get(r, target)
		require.Equal(t, http.StatusOK, rec.Code, target)
		var body map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), target)
		assert.Contains(t, body, "data", target)
	}
}

func TestSiteResponseNeverIncludesSecrets(t *testing.T) {
	rec := get(contentRouter(&stubContent{}), "/site")
	assert.Contains(t, rec.Body.String(), "Seafarers Crew")
	assert.NotContains(t, rec.Body.String(), "secret")
}

func TestPageUsesTheSlugFromThePathAndMissingPagesAre404(t *testing.T) {
	svc := &stubContent{}
	rec := get(contentRouter(svc), "/pages/about-us")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "about-us", svc.slug)

	svc.missingPage = true
	assert.Equal(t, http.StatusNotFound, get(contentRouter(svc), "/pages/draft").Code)
	assert.Equal(t, http.StatusNotFound, get(contentRouter(svc), "/news/unknown").Code)
}

func TestNewsListPassesPagingAndReturnsMeta(t *testing.T) {
	svc := &stubContent{articles: []model.News{{Title: "A", Slug: "a"}}}
	rec := get(contentRouter(svc), "/news?page=2&per_page=48")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, [2]int{2, 48}, [2]int{svc.page, svc.perPage})
	var body struct {
		Data []map[string]any `json:"data"`
		Meta struct{ Page, PerPage, Total int }
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 20, body.Meta.Total)
	assert.Equal(t, "a", body.Data[0]["slug"])

	get(contentRouter(svc), "/news?page=abc&per_page=")
	assert.Equal(t, [2]int{0, 0}, [2]int{svc.page, svc.perPage}, "unparseable values fall back to the service defaults")
}
