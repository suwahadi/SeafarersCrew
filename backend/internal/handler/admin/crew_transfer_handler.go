package admin

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// CrewExporter queues datagrid exports.
type CrewExporter interface {
	Enqueue(ctx context.Context, format string, f model.CrewFilter) (model.Job, error)
}

// CrewImporter validates and commits CSV imports.
type CrewImporter interface {
	Template() []byte
	Validate(ctx context.Context, data []byte) (service.ImportPreview, error)
	Commit(ctx context.Context, token string) (model.Job, error)
}

// UploadLimiter reports the per-file upload limit.
type UploadLimiter interface {
	MaxUploadBytes(ctx context.Context) (int64, error)
}

// CrewTransferHandler serves crew export and import.
type CrewTransferHandler struct {
	exporter CrewExporter
	importer CrewImporter
	limits   UploadLimiter
}

// NewCrewTransferHandler returns a CrewTransferHandler.
func NewCrewTransferHandler(exporter CrewExporter, importer CrewImporter, limits UploadLimiter) *CrewTransferHandler {
	return &CrewTransferHandler{exporter: exporter, importer: importer, limits: limits}
}

// Export handles POST /crews/export?format=csv|xlsx&<datagrid filters>.
func (h *CrewTransferHandler) Export(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f, errs := request.ParseCrewFilter(q)
	if len(errs) > 0 {
		resource.Error(w, r, apperr.Validation(errs))
		return
	}
	j, err := h.exporter.Enqueue(r.Context(), q.Get("format"), f)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	Accepted(w, j)
}

// Template handles GET /crews/import-template.
func (h *CrewTransferHandler) Template(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="crew-import-template.csv"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(h.importer.Template())
}

// Validate handles POST /crews/import/validate (multipart field "file").
func (h *CrewTransferHandler) Validate(w http.ResponseWriter, r *http.Request) {
	limit, err := h.limits.MaxUploadBytes(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit+64<<10)
	file, _, err := r.FormFile("file")
	if err != nil {
		resource.Error(w, r, uploadError(err))
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		resource.Error(w, r, uploadError(err))
		return
	}
	if int64(len(data)) > limit {
		resource.Error(w, r, apperr.ErrPayloadTooLarge)
		return
	}
	preview, err := h.importer.Validate(r.Context(), data)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, preview)
}

type importCommit struct {
	Token string `json:"token"`
}

// Commit handles POST /crews/import/commit.
func (h *CrewTransferHandler) Commit(w http.ResponseWriter, r *http.Request) {
	var in importCommit
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	j, err := h.importer.Commit(r.Context(), in.Token)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	Accepted(w, j)
}

func uploadError(err error) error {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return apperr.ErrPayloadTooLarge
	}
	return apperr.Validation(map[string]string{"file": "A CSV file is required."})
}
