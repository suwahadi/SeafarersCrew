package admin

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

// PageService is what the pages controller needs.
type PageService interface {
	List(ctx context.Context, q repository.ListQuery) (repository.Page[model.Page], error)
	Get(ctx context.Context, id uuid.UUID) (service.PageDetail, error)
	Create(ctx context.Context, in model.Page) (service.PageDetail, error)
	Update(ctx context.Context, id uuid.UUID, in model.Page) (service.PageDetail, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Reorder(ctx context.Context, ids []uuid.UUID) error
}

// PageHandler serves the page builder CRUD.
type PageHandler struct {
	svc     PageService
	present func(service.PageDetail) resource.Page
}

// NewPageHandler returns a PageHandler.
func NewPageHandler(svc PageService, urls resource.MediaURLer) *PageHandler {
	return &PageHandler{svc: svc, present: resource.NewPage(urls)}
}

var pageFilters = map[string]Filter{
	"status":    {Kind: FilterEnum, Options: []string{model.StatusDraft, model.StatusPublished}},
	"is_system": {Kind: FilterBool},
}

// Index handles GET /pages.
func (h *PageHandler) Index(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := request.ParseList(q)
	filters, err := parseMasterFilters(q, pageFilters)
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
	resource.List(w, resource.MapAll(page.Items, resource.NewPageListItem), resource.Meta{Page: params.Page, PerPage: params.PerPage, Total: page.Total})
}

// Show handles GET /pages/{id}.
func (h *PageHandler) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := masterPathID(w, r)
	if !ok {
		return
	}
	detail, err := h.svc.Get(r.Context(), id)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, h.present(detail))
}

// Store handles POST /pages.
func (h *PageHandler) Store(w http.ResponseWriter, r *http.Request) {
	var in request.Page
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	detail, err := h.svc.Create(r.Context(), in.Model())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusCreated, h.present(detail))
}

// Update handles PUT /pages/{id}.
func (h *PageHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := masterPathID(w, r)
	if !ok {
		return
	}
	var in request.Page
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	detail, err := h.svc.Update(r.Context(), id, in.Model())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, h.present(detail))
}

// Destroy handles DELETE /pages/{id}.
func (h *PageHandler) Destroy(w http.ResponseWriter, r *http.Request) {
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

// Reorder handles PATCH /pages/reorder.
func (h *PageHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	var in request.Reorder
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	if err := h.svc.Reorder(r.Context(), in.IDs); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}
