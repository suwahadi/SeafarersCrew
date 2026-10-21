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

// VacancyService is what the vacancy controller needs.
type VacancyService interface {
	Create(ctx context.Context, in model.Vacancy) (model.Vacancy, error)
	Get(ctx context.Context, publicID string) (model.Vacancy, error)
	List(ctx context.Context, f model.VacancyFilter, page, perPage int) (repository.Page[model.Vacancy], error)
	Update(ctx context.Context, publicID string, in model.Vacancy) (model.Vacancy, error)
	SetStatus(ctx context.Context, publicID, status string) (model.Vacancy, error)
	Delete(ctx context.Context, publicID string) error
}

// VacancyHandler serves job vacancies.
type VacancyHandler struct {
	vacancies VacancyService
}

// NewVacancyHandler returns a VacancyHandler.
func NewVacancyHandler(vacancies VacancyService) *VacancyHandler {
	return &VacancyHandler{vacancies: vacancies}
}

func vacancyID(r *http.Request) string { return chi.URLParam(r, "vacancy") }

// Index handles GET /vacancies.
func (h *VacancyHandler) Index(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f, errs := request.ParseVacancyFilter(q)
	if len(errs) > 0 {
		resource.Error(w, r, apperr.Validation(errs))
		return
	}
	p := request.ParseList(q)
	page, err := h.vacancies.List(r.Context(), f, p.Page, p.PerPage)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.List(w, resource.NewVacancies(page.Items), resource.Meta{Page: p.Page, PerPage: p.PerPage, Total: page.Total})
}

// Show handles GET /vacancies/{vacancy}.
func (h *VacancyHandler) Show(w http.ResponseWriter, r *http.Request) {
	v, err := h.vacancies.Get(r.Context(), vacancyID(r))
	respondVacancy(w, r, http.StatusOK, v, err)
}

// Store handles POST /vacancies.
func (h *VacancyHandler) Store(w http.ResponseWriter, r *http.Request) {
	var in request.Vacancy
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	v, err := h.vacancies.Create(r.Context(), in.Model())
	respondVacancy(w, r, http.StatusCreated, v, err)
}

// Update handles PUT /vacancies/{vacancy}.
func (h *VacancyHandler) Update(w http.ResponseWriter, r *http.Request) {
	var in request.Vacancy
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	v, err := h.vacancies.Update(r.Context(), vacancyID(r), in.Model())
	respondVacancy(w, r, http.StatusOK, v, err)
}

// UpdateStatus handles PATCH /vacancies/{vacancy}/status.
func (h *VacancyHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	var in request.VacancyStatus
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	v, err := h.vacancies.SetStatus(r.Context(), vacancyID(r), in.Status)
	respondVacancy(w, r, http.StatusOK, v, err)
}

// Destroy handles DELETE /vacancies/{vacancy}.
func (h *VacancyHandler) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.vacancies.Delete(r.Context(), vacancyID(r)); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}

func respondVacancy(w http.ResponseWriter, r *http.Request, status int, v model.Vacancy, err error) {
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, status, resource.NewVacancy(v))
}
