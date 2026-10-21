package public

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"

	"github.com/suwahadi/seafarerscrew/backend/internal/middleware"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

const (
	multipartOverhead = 1 << 20
	multipartMemory   = 1 << 20
	turnstileField    = "cf-turnstile-response"
	honeypotField     = "website"
)

// ApplicationService is what the application controller needs.
type ApplicationService interface {
	Limits(ctx context.Context) (service.ApplicationLimits, error)
	Submit(ctx context.Context, in service.ApplicationInput) (string, error)
}

// ApplicationHandler receives job applications.
type ApplicationHandler struct {
	applications ApplicationService
}

// NewApplicationHandler returns an ApplicationHandler.
func NewApplicationHandler(applications ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{applications: applications}
}

// Store handles POST /applications as multipart/form-data.
func (h *ApplicationHandler) Store(w http.ResponseWriter, r *http.Request) {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "multipart/form-data" {
		resource.Error(w, r, apperr.New(http.StatusUnsupportedMediaType, "unsupported_media_type", "The request must be multipart/form-data."))
		return
	}
	limits, err := h.applications.Limits(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(limits.MaxDocuments+1)*limits.MaxFileBytes+multipartOverhead)
	if err := r.ParseMultipartForm(multipartMemory); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			resource.Error(w, r, apperr.ErrPayloadTooLarge)
			return
		}
		resource.Error(w, r, apperr.ErrBadRequest.Wrap(err))
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()
	id, err := h.applications.Submit(r.Context(), applicationInput(r))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusCreated, map[string]string{"application_id": id})
}

func applicationInput(r *http.Request) service.ApplicationInput {
	form := r.MultipartForm
	value := func(name string) string {
		if v := form.Value[name]; len(v) > 0 {
			return v[0]
		}
		return ""
	}
	return service.ApplicationInput{
		Vacancy: value("vacancy"), FullName: value("full_name"), Email: value("email"), Phone: value("phone"),
		DateOfBirth: value("date_of_birth"), Nationality: value("nationality"), RankAppliedID: value("rank_applied_id"),
		YearsExperience: value("years_experience"), Message: value("message"), Consent: value("consent"),
		Website: value(honeypotField), TurnstileToken: value(turnstileField), IP: middleware.ClientIP(r.Context()),
		CV: applicationFiles(form.File["cv"]), Documents: applicationFiles(append(form.File["documents"], form.File["documents[]"]...)),
	}
}

func applicationFiles(headers []*multipart.FileHeader) []service.ApplicationFile {
	out := make([]service.ApplicationFile, 0, len(headers))
	for _, h := range headers {
		out = append(out, service.ApplicationFile{
			Filename: h.Filename,
			Size:     h.Size,
			Open:     func() (io.ReadCloser, error) { return h.Open() },
		})
	}
	return out
}
