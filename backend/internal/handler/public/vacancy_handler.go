package public

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// VacancyService is what the public vacancy controller needs.
type VacancyService interface {
	PublicList(ctx context.Context, q model.PublicVacancyQuery) (repository.Page[model.Vacancy], error)
	PublicDetail(ctx context.Context, slug string) (service.PublicVacancy, error)
	PublicFilters(ctx context.Context) (model.VacancyFilterOptions, error)
}

// VacancyHandler serves open vacancies to the website.
type VacancyHandler struct {
	vacancies VacancyService
}

// NewVacancyHandler returns a VacancyHandler.
func NewVacancyHandler(vacancies VacancyService) *VacancyHandler {
	return &VacancyHandler{vacancies: vacancies}
}

// Index handles GET /vacancies.
func (h *VacancyHandler) Index(w http.ResponseWriter, r *http.Request) {
	q, errs := request.ParsePublicVacancyQuery(r.URL.Query())
	if len(errs) > 0 {
		resource.Error(w, r, apperr.Validation(errs))
		return
	}
	page, err := h.vacancies.PublicList(r.Context(), q)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.List(w, resource.NewPublicVacancyCards(page.Items), resource.Meta{Page: q.Page, PerPage: q.PerPage, Total: page.Total})
}

// Show handles GET /vacancies/{slug}.
func (h *VacancyHandler) Show(w http.ResponseWriter, r *http.Request) {
	v, err := h.vacancies.PublicDetail(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewPublicVacancy(v))
}

// Filters handles GET /vacancy-filters.
func (h *VacancyHandler) Filters(w http.ResponseWriter, r *http.Request) {
	o, err := h.vacancies.PublicFilters(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewPublicVacancyFilters(o))
}
