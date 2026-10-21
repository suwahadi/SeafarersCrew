package admin

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/middleware"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// ApplicantService is what the applicant controller needs.
type ApplicantService interface {
	Get(ctx context.Context, publicID string) (model.Applicant, error)
	List(ctx context.Context, f model.ApplicantFilter, page, perPage int) (repository.Page[model.Applicant], error)
	StageCounts(ctx context.Context, f model.ApplicantFilter) (map[string]int, error)
	ChangeStage(ctx context.Context, publicID string, adminID uuid.UUID, stage, note string) (model.Applicant, error)
	Delete(ctx context.Context, publicID string) error
}

// ApplicantHandler serves the applicant pipeline.
type ApplicantHandler struct {
	applicants ApplicantService
}

// NewApplicantHandler returns an ApplicantHandler.
func NewApplicantHandler(applicants ApplicantService) *ApplicantHandler {
	return &ApplicantHandler{applicants: applicants}
}

func applicantID(r *http.Request) string { return chi.URLParam(r, "applicant") }

// Index handles GET /applicants.
func (h *ApplicantHandler) Index(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f, errs := request.ParseApplicantFilter(q)
	if len(errs) > 0 {
		resource.Error(w, r, apperr.Validation(errs))
		return
	}
	p := request.ParseList(q)
	page, err := h.applicants.List(r.Context(), f, p.Page, p.PerPage)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.List(w, resource.NewApplicants(page.Items), resource.Meta{Page: p.Page, PerPage: p.PerPage, Total: page.Total})
}

// StageCounts handles GET /applicants/stage-counts.
func (h *ApplicantHandler) StageCounts(w http.ResponseWriter, r *http.Request) {
	f, errs := request.ParseApplicantFilter(r.URL.Query())
	if len(errs) > 0 {
		resource.Error(w, r, apperr.Validation(errs))
		return
	}
	counts, err := h.applicants.StageCounts(r.Context(), f)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, counts)
}

// Show handles GET /applicants/{applicant}.
func (h *ApplicantHandler) Show(w http.ResponseWriter, r *http.Request) {
	a, err := h.applicants.Get(r.Context(), applicantID(r))
	respondApplicant(w, r, a, err)
}

// UpdateStage handles PATCH /applicants/{applicant}/stage.
func (h *ApplicantHandler) UpdateStage(w http.ResponseWriter, r *http.Request) {
	var in request.ApplicantStage
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	actor, _ := middleware.Admin(r.Context())
	a, err := h.applicants.ChangeStage(r.Context(), applicantID(r), actor.ID, in.Stage, in.Note)
	respondApplicant(w, r, a, err)
}

// Destroy handles DELETE /applicants/{applicant}.
func (h *ApplicantHandler) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.applicants.Delete(r.Context(), applicantID(r)); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}

func respondApplicant(w http.ResponseWriter, r *http.Request, a model.Applicant, err error) {
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewApplicantDetail(a))
}
