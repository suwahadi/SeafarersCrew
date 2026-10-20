package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// JobReader loads job state.
type JobReader interface {
	Get(ctx context.Context, id uuid.UUID) (model.Job, error)
}

// JobHandler serves GET /jobs/{id}.
type JobHandler struct {
	jobs JobReader
}

// NewJobHandler returns a JobHandler.
func NewJobHandler(jobs JobReader) *JobHandler {
	return &JobHandler{jobs: jobs}
}

type jobResource struct {
	ID         string          `json:"id"`
	Kind       string          `json:"kind"`
	Status     string          `json:"status"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	Attempts   int             `json:"attempts"`
	CreatedAt  time.Time       `json:"created_at"`
	FinishedAt *time.Time      `json:"finished_at,omitempty"`
}

// Show handles GET /jobs/{id}.
func (h *JobHandler) Show(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		resource.Error(w, r, apperr.NotFound("Job"))
		return
	}
	j, err := h.jobs.Get(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		resource.Error(w, r, apperr.NotFound("Job"))
		return
	}
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, jobResource{
		ID: j.ID.String(), Kind: j.Kind, Status: j.Status, Result: j.Result, Error: j.Error,
		Attempts: j.Attempts, CreatedAt: j.CreatedAt, FinishedAt: j.FinishedAt,
	})
}

// Accepted writes 202 with the queued job.
func Accepted(w http.ResponseWriter, j model.Job) {
	resource.JSON(w, http.StatusAccepted, jobResource{ID: j.ID.String(), Kind: j.Kind, Status: j.Status, CreatedAt: j.CreatedAt})
}
