package admin

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
)

// NewsService is what the news controller needs.
type NewsService interface {
	List(ctx context.Context, q repository.ListQuery) (repository.Page[model.News], error)
	Get(ctx context.Context, id uuid.UUID) (model.News, error)
	Create(ctx context.Context, in model.News) (model.News, error)
	Update(ctx context.Context, id uuid.UUID, in model.News) (model.News, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// NewsHandler serves the news CRUD.
type NewsHandler struct {
	svc     NewsService
	item    func(model.News) resource.NewsListItem
	present func(model.News) resource.News
}

// NewNewsHandler returns a NewsHandler.
func NewNewsHandler(svc NewsService, urls resource.MediaURLer, now func() time.Time) *NewsHandler {
	return &NewsHandler{svc: svc, item: resource.NewNewsListItem(urls, now), present: resource.NewNews(urls, now)}
}

var newsFilters = map[string]Filter{
	"status": {Kind: FilterEnum, Options: []string{model.StatusDraft, model.StatusPublished}},
}

// Index handles GET /news.
func (h *NewsHandler) Index(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := request.ParseList(q)
	filters, err := parseMasterFilters(q, newsFilters)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	page, err := h.svc.List(r.Context(), repository.ListQuery{
		Page: params.Page, PerPage: params.PerPage, Sort: params.Sort, Desc: params.Desc, Q: params.Q, Filters: filters,
	})
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.List(w, resource.MapAll(page.Items, h.item), resource.Meta{Page: params.Page, PerPage: params.PerPage, Total: page.Total})
}

// Show handles GET /news/{id}.
func (h *NewsHandler) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := masterPathID(w, r)
	if !ok {
		return
	}
	n, err := h.svc.Get(r.Context(), id)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, h.present(n))
}

// Store handles POST /news.
func (h *NewsHandler) Store(w http.ResponseWriter, r *http.Request) {
	var in request.News
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	n, err := h.svc.Create(r.Context(), in.Model())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusCreated, h.present(n))
}

// Update handles PUT /news/{id}.
func (h *NewsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := masterPathID(w, r)
	if !ok {
		return
	}
	var in request.News
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	n, err := h.svc.Update(r.Context(), id, in.Model())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, h.present(n))
}

// Destroy handles DELETE /news/{id}.
func (h *NewsHandler) Destroy(w http.ResponseWriter, r *http.Request) {
	id, ok := masterPathID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}
