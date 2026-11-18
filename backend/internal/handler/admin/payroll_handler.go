package admin

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/filecheck"
)

// PayrollService is what the payroll controller needs.
type PayrollService interface {
	CreatePeriod(ctx context.Context, in model.PayrollPeriod) (model.PayrollPeriod, error)
	GetPeriod(ctx context.Context, publicID string) (model.PayrollPeriod, error)
	ListPeriods(ctx context.Context, q string, page, perPage int) (repository.Page[model.PayrollPeriod], error)
	UpdatePeriod(ctx context.Context, publicID string, in model.PayrollPeriod) (model.PayrollPeriod, error)
	DeletePeriod(ctx context.Context, publicID string) error
	Lines(ctx context.Context, publicID string, f model.PayrollLineFilter, page, perPage int) (repository.Page[model.PayrollLine], model.PayrollTotals, error)
	Facets(ctx context.Context, publicID string) (model.PayrollFacets, error)
	CreateLine(ctx context.Context, periodPublicID string, in model.PayrollLineInput) (model.PayrollLine, error)
	UpdateLine(ctx context.Context, id uuid.UUID, in model.PayrollLineInput) (model.PayrollLine, error)
	DeleteLine(ctx context.Context, id uuid.UUID) error
}

// PayrollImporter validates and commits payroll workbooks.
type PayrollImporter interface {
	Validate(ctx context.Context, periodPublicID string, data []byte) (service.PayrollPreview, error)
	Commit(ctx context.Context, periodPublicID, token string) (model.Job, error)
}

// PayrollExporter queues payroll workbook exports.
type PayrollExporter interface {
	Enqueue(ctx context.Context, periodPublicID string, f model.PayrollLineFilter) (model.Job, error)
}

// PayrollHandler serves payroll periods, their lines, imports and exports.
type PayrollHandler struct {
	payroll  PayrollService
	importer PayrollImporter
	exporter PayrollExporter
	limits   UploadLimiter
}

// NewPayrollHandler returns a PayrollHandler.
func NewPayrollHandler(payroll PayrollService, importer PayrollImporter, exporter PayrollExporter, limits UploadLimiter) *PayrollHandler {
	return &PayrollHandler{payroll: payroll, importer: importer, exporter: exporter, limits: limits}
}

func periodID(r *http.Request) string { return chi.URLParam(r, "period") }

func payrollLineID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperr.NotFound("Payroll line")
	}
	return id, nil
}

func respondPayrollPeriod(w http.ResponseWriter, r *http.Request, status int, p model.PayrollPeriod, err error) {
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, status, resource.NewPayrollPeriod(p))
}

func respondPayrollLine(w http.ResponseWriter, r *http.Request, status int, l model.PayrollLine, err error) {
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, status, resource.NewPayrollLine(l))
}

// Index handles GET /payroll-periods.
func (h *PayrollHandler) Index(w http.ResponseWriter, r *http.Request) {
	p := request.ParseList(r.URL.Query())
	page, err := h.payroll.ListPeriods(r.Context(), p.Q, p.Page, p.PerPage)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.List(w, resource.NewPayrollPeriods(page.Items), resource.Meta{Page: p.Page, PerPage: p.PerPage, Total: page.Total})
}

// Store handles POST /payroll-periods.
func (h *PayrollHandler) Store(w http.ResponseWriter, r *http.Request) {
	var in request.PayrollPeriod
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	p, err := h.payroll.CreatePeriod(r.Context(), in.Model())
	respondPayrollPeriod(w, r, http.StatusCreated, p, err)
}

// Show handles GET /payroll-periods/{period}.
func (h *PayrollHandler) Show(w http.ResponseWriter, r *http.Request) {
	p, err := h.payroll.GetPeriod(r.Context(), periodID(r))
	respondPayrollPeriod(w, r, http.StatusOK, p, err)
}

// Update handles PUT /payroll-periods/{period}.
func (h *PayrollHandler) Update(w http.ResponseWriter, r *http.Request) {
	var in request.PayrollPeriod
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	p, err := h.payroll.UpdatePeriod(r.Context(), periodID(r), in.Model())
	respondPayrollPeriod(w, r, http.StatusOK, p, err)
}

// Destroy handles DELETE /payroll-periods/{period}.
func (h *PayrollHandler) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.payroll.DeletePeriod(r.Context(), periodID(r)); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}

// Lines handles GET /payroll-periods/{period}/lines.
func (h *PayrollHandler) Lines(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f, errs := request.ParsePayrollLineFilter(q)
	if len(errs) > 0 {
		resource.Error(w, r, apperr.Validation(errs))
		return
	}
	p := request.ParseList(q)
	page, totals, err := h.payroll.Lines(r.Context(), periodID(r), f, p.Page, p.PerPage)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.ListWithTotals(w, resource.NewPayrollLines(page.Items), resource.Meta{Page: p.Page, PerPage: p.PerPage, Total: page.Total},
		resource.NewPayrollTotals(totals))
}

// Facets handles GET /payroll-periods/{period}/facets.
func (h *PayrollHandler) Facets(w http.ResponseWriter, r *http.Request) {
	f, err := h.payroll.Facets(r.Context(), periodID(r))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewPayrollFacets(f))
}

// StoreLine handles POST /payroll-periods/{period}/lines.
func (h *PayrollHandler) StoreLine(w http.ResponseWriter, r *http.Request) {
	var in request.PayrollLine
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	l, err := h.payroll.CreateLine(r.Context(), periodID(r), in.Input())
	respondPayrollLine(w, r, http.StatusCreated, l, err)
}

// UpdateLine handles PUT /payroll-lines/{id}.
func (h *PayrollHandler) UpdateLine(w http.ResponseWriter, r *http.Request) {
	id, err := payrollLineID(r)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	var in request.PayrollLine
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	l, err := h.payroll.UpdateLine(r.Context(), id, in.Input())
	respondPayrollLine(w, r, http.StatusOK, l, err)
}

// DestroyLine handles DELETE /payroll-lines/{id}.
func (h *PayrollHandler) DestroyLine(w http.ResponseWriter, r *http.Request) {
	id, err := payrollLineID(r)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	if err := h.payroll.DeleteLine(r.Context(), id); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}

// ValidateImport handles POST /payroll-periods/{period}/import/validate (multipart field "file").
func (h *PayrollHandler) ValidateImport(w http.ResponseWriter, r *http.Request) {
	limit, err := h.limits.MaxUploadBytes(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit+64<<10)
	file, header, err := r.FormFile("file")
	if err != nil {
		resource.Error(w, r, workbookUploadError(err))
		return
	}
	defer func() { _ = file.Close() }()
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(header.Filename), "."))
	if err := filecheck.CheckExtension(filecheck.ContextPayrollImport, ext); err != nil {
		resource.Error(w, r, err)
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		resource.Error(w, r, workbookUploadError(err))
		return
	}
	if int64(len(data)) > limit {
		resource.Error(w, r, apperr.ErrPayloadTooLarge)
		return
	}
	preview, err := h.importer.Validate(r.Context(), periodID(r), data)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, preview)
}

// CommitImport handles POST /payroll-periods/{period}/import/commit.
func (h *PayrollHandler) CommitImport(w http.ResponseWriter, r *http.Request) {
	var in importCommit
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	j, err := h.importer.Commit(r.Context(), periodID(r), in.Token)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	Accepted(w, j)
}

// Export handles POST /payroll-periods/{period}/export?<lines filters>.
func (h *PayrollHandler) Export(w http.ResponseWriter, r *http.Request) {
	f, errs := request.ParsePayrollLineFilter(r.URL.Query())
	if len(errs) > 0 {
		resource.Error(w, r, apperr.Validation(errs))
		return
	}
	j, err := h.exporter.Enqueue(r.Context(), periodID(r), f)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	Accepted(w, j)
}

func workbookUploadError(err error) error {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return apperr.ErrPayloadTooLarge
	}
	return apperr.Validation(map[string]string{"file": "An Excel (.xlsx) file is required."})
}
