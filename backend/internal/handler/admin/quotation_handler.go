package admin

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// QuotationService is what the quotation controller needs.
type QuotationService interface {
	List(ctx context.Context, f model.QuotationFilter) (repository.Page[model.Quotation], error)
	Get(ctx context.Context, publicID string) (model.Quotation, error)
	Create(ctx context.Context, q model.Quotation) (model.Quotation, error)
	Update(ctx context.Context, publicID string, q model.Quotation) (model.Quotation, error)
	ChangeStatus(ctx context.Context, publicID, to string) (model.Quotation, error)
	Delete(ctx context.Context, publicID string) error
}

// QuotationHandler serves quotations.
type QuotationHandler struct {
	quotations QuotationService
}

// NewQuotationHandler returns a QuotationHandler.
func NewQuotationHandler(quotations QuotationService) *QuotationHandler {
	return &QuotationHandler{quotations: quotations}
}

func quotationID(r *http.Request) string { return chi.URLParam(r, "id") }

// Index handles GET /quotations.
func (h *QuotationHandler) Index(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f, errs := request.ParseQuotationFilter(q)
	if len(errs) > 0 {
		resource.Error(w, r, apperr.Validation(errs))
		return
	}
	p := request.ParseList(q)
	page, err := h.quotations.List(r.Context(), f)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.List(w, resource.MapAll(page.Items, resource.NewQuotationRow), resource.Meta{Page: p.Page, PerPage: p.PerPage, Total: page.Total})
}

// Show handles GET /quotations/{id}.
func (h *QuotationHandler) Show(w http.ResponseWriter, r *http.Request) {
	q, err := h.quotations.Get(r.Context(), quotationID(r))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewQuotation(q))
}

// Store handles POST /quotations.
func (h *QuotationHandler) Store(w http.ResponseWriter, r *http.Request) {
	var in request.Quotation
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	q, err := h.quotations.Create(r.Context(), in.Model())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusCreated, resource.NewQuotation(q))
}

// Update handles PUT /quotations/{id}.
func (h *QuotationHandler) Update(w http.ResponseWriter, r *http.Request) {
	var in request.Quotation
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	q, err := h.quotations.Update(r.Context(), quotationID(r), in.Model())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewQuotation(q))
}

// Status handles PATCH /quotations/{id}/status.
func (h *QuotationHandler) Status(w http.ResponseWriter, r *http.Request) {
	var in request.QuotationStatus
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	q, err := h.quotations.ChangeStatus(r.Context(), quotationID(r), in.Status)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewQuotation(q))
}

// Destroy handles DELETE /quotations/{id}.
func (h *QuotationHandler) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.quotations.Delete(r.Context(), quotationID(r)); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}
