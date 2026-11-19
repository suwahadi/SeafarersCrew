package admin

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// CrewCVImporter validates and commits single-CV imports.
type CrewCVImporter interface {
	Validate(ctx context.Context, filename string, data []byte) (service.CVPreview, error)
	Commit(ctx context.Context, in service.CVCommitInput) (service.CVCommitResult, error)
}

// CrewCVHandler serves the crew CV import endpoints.
type CrewCVHandler struct {
	importer CrewCVImporter
	limits   UploadLimiter
}

// NewCrewCVHandler builds a CrewCVHandler.
func NewCrewCVHandler(importer CrewCVImporter, limits UploadLimiter) *CrewCVHandler {
	return &CrewCVHandler{importer: importer, limits: limits}
}

func cvUploadError(err error) error {
	var maxErr *http.MaxBytesError
	if err != nil && errors.As(err, &maxErr) {
		return apperr.ErrPayloadTooLarge
	}
	return apperr.Validation(map[string]string{"file": "Save the CV as .xlsx and upload it again."})
}

// Validate handles POST /crews/cv-import/validate (multipart field "file").
func (h *CrewCVHandler) Validate(w http.ResponseWriter, r *http.Request) {
	limit, err := h.limits.MaxUploadBytes(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit+64<<10)
	file, header, err := r.FormFile("file")
	if err != nil {
		resource.Error(w, r, cvUploadError(err))
		return
	}
	defer func() { _ = file.Close() }()
	if !strings.EqualFold(path.Ext(header.Filename), ".xlsx") {
		resource.Error(w, r, cvUploadError(nil))
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		resource.Error(w, r, cvUploadError(err))
		return
	}
	if int64(len(data)) > limit {
		resource.Error(w, r, apperr.ErrPayloadTooLarge)
		return
	}
	preview, err := h.importer.Validate(r.Context(), header.Filename, data)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, preview)
}

// Commit handles POST /crews/cv-import/commit.
func (h *CrewCVHandler) Commit(w http.ResponseWriter, r *http.Request) {
	var in service.CVCommitInput
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	result, err := h.importer.Commit(r.Context(), in)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, result)
}
