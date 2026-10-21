package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type noMedia struct{}

func (noMedia) URL(string) string { return "" }

type stubPages struct {
	lastQuery repository.ListQuery
	created   model.Page
	updatedID uuid.UUID
	reordered []uuid.UUID
}

func (s *stubPages) List(_ context.Context, q repository.ListQuery) (repository.Page[model.Page], error) {
	s.lastQuery = q
	return repository.Page[model.Page]{Items: []model.Page{{Title: "About", Slug: "about"}}, Total: 7}, nil
}

func (s *stubPages) Get(context.Context, uuid.UUID) (service.PageDetail, error) {
	return service.PageDetail{}, apperr.NotFound("Page")
}

func (s *stubPages) Create(_ context.Context, in model.Page) (service.PageDetail, error) {
	s.created = in
	in.ID = uuid.New()
	return service.PageDetail{Page: in}, nil
}

func (s *stubPages) Update(_ context.Context, id uuid.UUID, in model.Page) (service.PageDetail, error) {
	s.updatedID = id
	return service.PageDetail{Page: in}, nil
}

func (s *stubPages) Delete(context.Context, uuid.UUID) error {
	return apperr.Conflict("system_page", "System pages cannot be deleted.")
}

func (s *stubPages) Reorder(_ context.Context, ids []uuid.UUID) error {
	s.reordered = ids
	return nil
}

func pageRouter(svc *stubPages) http.Handler {
	h := NewPageHandler(svc, noMedia{})
	r := chi.NewRouter()
	r.Get("/pages", h.Index)
	r.Post("/pages", h.Store)
	r.Patch("/pages/reorder", h.Reorder)
	r.Get("/pages/{id}", h.Show)
	r.Put("/pages/{id}", h.Update)
	r.Delete("/pages/{id}", h.Destroy)
	return r
}

func sendCMS(h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPageIndexParsesFiltersAndReturnsMeta(t *testing.T) {
	svc := &stubPages{}
	rec := sendCMS(pageRouter(svc), http.MethodGet, "/pages?status=published&is_system=false&q=ab&sort=title&per_page=10", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data []map[string]any `json:"data"`
		Meta struct{ Total int }
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 7, body.Meta.Total)
	assert.Equal(t, "/about", body.Data[0]["path"])
	assert.NotContains(t, body.Data[0], "blocks", "list rows omit the page content")
	assert.Equal(t, map[string]any{"status": "published", "is_system": false}, svc.lastQuery.Filters)

	assert.Equal(t, http.StatusUnprocessableEntity, sendCMS(pageRouter(svc), http.MethodGet, "/pages?status=archived", "").Code)
}

func TestPageStoreReturnsBlocksAndFieldLevelBlockErrors(t *testing.T) {
	svc := &stubPages{}
	rec := sendCMS(pageRouter(svc), http.MethodPost, "/pages", `{"title":"About","blocks":[{"id":"a","type":"rich_text","settings":{"html":"<p>x</p>"}}]}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created struct {
		Data struct {
			Blocks []map[string]any `json:"blocks"`
			Media  map[string]any   `json:"media"`
		}
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	require.Len(t, created.Data.Blocks, 1)
	assert.Equal(t, "rich_text", created.Data.Blocks[0]["type"])
	assert.Equal(t, map[string]any{"html": "<p>x</p>"}, created.Data.Blocks[0]["settings"])
	assert.NotNil(t, created.Data.Media)

	rec = sendCMS(pageRouter(svc), http.MethodPost, "/pages", `{"title":"About","blocks":[{"id":"a","type":"rich_text","settings":{"html":"x"}},{"id":"b","type":"gallery","settings":{"image_ids":["`+uuid.NewString()+`"],"columns":9}}]}`)
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var failed struct {
		Error struct{ Fields map[string]string }
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &failed))
	assert.Contains(t, failed.Error.Fields, "blocks.1.settings.columns")
}

func TestPageUpdateDeleteAndReorder(t *testing.T) {
	svc := &stubPages{}
	id := uuid.New()
	rec := sendCMS(pageRouter(svc), http.MethodPut, "/pages/"+id.String(), `{"title":"About","status":"published"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, id, svc.updatedID)

	rec = sendCMS(pageRouter(svc), http.MethodDelete, "/pages/"+id.String(), "")
	assert.Equal(t, http.StatusConflict, rec.Code)

	rec = sendCMS(pageRouter(svc), http.MethodGet, "/pages/not-a-uuid", "")
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = sendCMS(pageRouter(svc), http.MethodPatch, "/pages/reorder", `{"ids":["`+id.String()+`"]}`)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, []uuid.UUID{id}, svc.reordered)
	assert.Equal(t, http.StatusUnprocessableEntity, sendCMS(pageRouter(svc), http.MethodPatch, "/pages/reorder", `{"ids":[]}`).Code)
}

type stubNews struct {
	created model.News
}

func (s *stubNews) List(context.Context, repository.ListQuery) (repository.Page[model.News], error) {
	return repository.Page[model.News]{Items: []model.News{{Title: "A", Status: model.StatusPublished, PublishedAt: ptrTime(time.Now().Add(-time.Hour))}}, Total: 1}, nil
}

func (s *stubNews) Get(context.Context, uuid.UUID) (model.News, error) {
	return model.News{}, apperr.NotFound("Article")
}

func (s *stubNews) Create(_ context.Context, in model.News) (model.News, error) {
	s.created = in
	return in, nil
}

func (s *stubNews) Update(_ context.Context, _ uuid.UUID, in model.News) (model.News, error) {
	return in, nil
}
func (s *stubNews) Delete(context.Context, uuid.UUID) error { return nil }

func ptrTime(t time.Time) *time.Time { return &t }

func TestNewsHandlerListsWithLiveFlagAndDecodesPublishDate(t *testing.T) {
	svc := &stubNews{}
	h := NewNewsHandler(svc, noMedia{}, time.Now)
	r := chi.NewRouter()
	r.Get("/news", h.Index)
	r.Post("/news", h.Store)

	rec := sendCMS(r, http.MethodGet, "/news?status=published", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var list struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	assert.Equal(t, true, list.Data[0]["is_live"])
	assert.NotContains(t, list.Data[0], "body_html")

	rec = sendCMS(r, http.MethodPost, "/news", `{"title":"Crew change","status":"published","published_at":"2026-12-01T08:00:00Z"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.NotNil(t, svc.created.PublishedAt)
	assert.Equal(t, 2026, svc.created.PublishedAt.Year())

	assert.Equal(t, http.StatusUnprocessableEntity, sendCMS(r, http.MethodPost, "/news", `{"title":"x","excerpt":"`+strings.Repeat("e", 301)+`"}`).Code)
}

type stubMenus struct {
	menu  string
	items []model.MenuItem
}

func (s *stubMenus) Get(context.Context) (model.Menus, error) { return model.Menus{}, nil }

func (s *stubMenus) Replace(_ context.Context, menu string, items []model.MenuItem) (model.Menus, error) {
	s.menu, s.items = menu, items
	return model.Menus{Header: items}, nil
}

func TestMenuReplaceDecodesNestedItemsAndReturnsTheTree(t *testing.T) {
	svc := &stubMenus{}
	h := NewMenuHandler(svc)
	r := chi.NewRouter()
	r.Get("/menus", h.Index)
	r.Put("/menus/{menu}", h.Replace)

	rec := sendCMS(r, http.MethodPut, "/menus/header", `{"items":[{"label":"About","url":"/about","children":[{"label":"Safety","url":"/safety"}]}]}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "header", svc.menu)
	require.Len(t, svc.items, 1)
	assert.Len(t, svc.items[0].Children, 1)
	var body struct {
		Data struct {
			Header []map[string]any `json:"header"`
			Footer []any            `json:"footer"`
		}
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "About", body.Data.Header[0]["label"])
	assert.Equal(t, []any{}, body.Data.Footer)

	assert.Equal(t, http.StatusUnprocessableEntity, sendCMS(r, http.MethodPut, "/menus/footer", `{"items":[{"label":"X","url":"javascript:x"}]}`).Code)
	assert.Equal(t, http.StatusOK, sendCMS(r, http.MethodGet, "/menus", "").Code)
}

type stubGallery struct {
	stubRanksLike
	bulk []model.GalleryImage
}

type stubRanksLike struct{}

func (stubRanksLike) List(context.Context, repository.ListQuery) (repository.Page[model.GalleryImage], error) {
	return repository.Page[model.GalleryImage]{}, nil
}
func (stubRanksLike) Get(context.Context, uuid.UUID) (model.GalleryImage, error) {
	return model.GalleryImage{}, nil
}
func (stubRanksLike) Create(_ context.Context, g model.GalleryImage) (model.GalleryImage, error) {
	return g, nil
}
func (stubRanksLike) Update(_ context.Context, _ uuid.UUID, g model.GalleryImage) (model.GalleryImage, error) {
	return g, nil
}
func (stubRanksLike) Delete(context.Context, uuid.UUID) error    { return nil }
func (stubRanksLike) Reorder(context.Context, []uuid.UUID) error { return nil }

func (s *stubGallery) BulkCreate(_ context.Context, items []model.GalleryImage) ([]model.GalleryImage, error) {
	s.bulk = items
	return items, nil
}

func TestGalleryBulkCreatesManyImagesFromFileIDs(t *testing.T) {
	svc := &stubGallery{}
	h := NewGalleryHandler(svc, noMedia{})
	r := chi.NewRouter()
	r.Post("/gallery/bulk", h.Bulk)
	r.Get("/gallery", h.List)

	id := uuid.NewString()
	rec := sendCMS(r, http.MethodPost, "/gallery/bulk", `{"items":[{"image_file_id":"`+id+`","alt_text":"Deck"},{"image_file_id":"`+id+`"}],"show_on_home":true}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Len(t, svc.bulk, 2)
	assert.True(t, svc.bulk[0].ShowOnHome)
	assert.Equal(t, "Deck", svc.bulk[0].AltText)

	assert.Equal(t, http.StatusUnprocessableEntity, sendCMS(r, http.MethodPost, "/gallery/bulk", `{"items":[]}`).Code)
	assert.Equal(t, http.StatusOK, sendCMS(r, http.MethodGet, "/gallery?show_on_home=true", "").Code)
}
