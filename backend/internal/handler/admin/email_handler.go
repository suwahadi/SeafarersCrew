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
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// EmailService is what the email controller needs.
type EmailService interface {
	Send(ctx context.Context, in service.EmailSendInput) (model.EmailLog, model.Job, error)
	Resend(ctx context.Context, id uuid.UUID) (model.EmailLog, model.Job, error)
	ListLogs(ctx context.Context, f model.EmailLogFilter, page, perPage int) (repository.Page[model.EmailLog], error)
	Templates(ctx context.Context) ([]model.EmailTemplate, error)
	Template(ctx context.Context, code string) (model.EmailTemplate, error)
	UpdateTemplate(ctx context.Context, code, subject, body string) (model.EmailTemplate, error)
	PreviewTemplate(ctx context.Context, code, subject, body string, vars map[string]string) (service.EmailPreview, error)
}

// ConnectionTester runs Settings "Test Connection" checks.
type ConnectionTester interface {
	Test(ctx context.Context, group string, admin model.Admin) (service.EmailConnectionResult, error)
}

// EmailHandler serves email dispatch, email logs, email templates and connection tests.
type EmailHandler struct {
	emails EmailService
	tester ConnectionTester
}

// NewEmailHandler returns an EmailHandler.
func NewEmailHandler(emails EmailService, tester ConnectionTester) *EmailHandler {
	return &EmailHandler{emails: emails, tester: tester}
}

// Send handles POST /emails/send.
func (h *EmailHandler) Send(w http.ResponseWriter, r *http.Request) {
	var in request.SendEmail
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	log, j, err := h.emails.Send(r.Context(), service.EmailSendInput{
		TemplateCode: in.TemplateCode, To: in.To, CC: in.CC, Subject: in.Subject, BodyHTML: in.BodyHTML,
		Variables: in.Variables, RelatedType: in.RelatedType, RelatedID: in.RelatedUUID(), AttachmentFileIDs: in.Attachments(),
	})
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusAccepted, resource.QueuedEmail{Log: resource.NewEmailLog(log), JobID: j.ID})
}

// Logs handles GET /email-logs?related_type=&related_id=&status=.
func (h *EmailHandler) Logs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := request.ParseList(q)
	e := request.Errors{}
	f := model.EmailLogFilter{RelatedType: q.Get("related_type"), Status: q.Get("status")}
	f.RelatedID = e.UUID("related_id", q.Get("related_id"))
	e.OneOf("status", f.Status, model.EmailQueued, model.EmailSent, model.EmailFailed)
	e.OneOf("related_type", f.RelatedType, service.EmailRelatedTypes...)
	if len(e) > 0 {
		resource.Error(w, r, apperr.Validation(e))
		return
	}
	page, err := h.emails.ListLogs(r.Context(), f, p.Page, p.PerPage)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	items := make([]resource.EmailLog, len(page.Items))
	for i, l := range page.Items {
		items[i] = resource.NewEmailLog(l)
	}
	resource.List(w, items, resource.Meta{Page: p.Page, PerPage: p.PerPage, Total: page.Total})
}

// Resend handles POST /email-logs/{id}/resend.
func (h *EmailHandler) Resend(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		resource.Error(w, r, apperr.NotFound("Email log"))
		return
	}
	log, j, err := h.emails.Resend(r.Context(), id)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusAccepted, resource.QueuedEmail{Log: resource.NewEmailLog(log), JobID: j.ID})
}

// Templates handles GET /email-templates.
func (h *EmailHandler) Templates(w http.ResponseWriter, r *http.Request) {
	list, err := h.emails.Templates(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	out := make([]resource.EmailTemplate, len(list))
	for i, t := range list {
		out[i] = resource.NewEmailTemplate(t, false, nil)
	}
	resource.JSON(w, http.StatusOK, out)
}

// Template handles GET /email-templates/{code}.
func (h *EmailHandler) Template(w http.ResponseWriter, r *http.Request) {
	t, err := h.emails.Template(r.Context(), chi.URLParam(r, "code"))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewEmailTemplate(t, true, service.EmailAvailableTags(t.Code)))
}

// UpdateTemplate handles PUT /email-templates/{code}.
func (h *EmailHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	var in request.EmailTemplate
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	t, err := h.emails.UpdateTemplate(r.Context(), chi.URLParam(r, "code"), in.Subject, in.BodyHTML)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewEmailTemplate(t, true, service.EmailAvailableTags(t.Code)))
}

// Preview handles POST /email-templates/{code}/preview.
func (h *EmailHandler) Preview(w http.ResponseWriter, r *http.Request) {
	var in request.EmailPreview
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	p, err := h.emails.PreviewTemplate(r.Context(), chi.URLParam(r, "code"), in.Subject, in.BodyHTML, in.Variables)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, p)
}

// TestConnection handles POST /settings/{group}/test.
func (h *EmailHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	admin, _ := middleware.Admin(r.Context())
	res, err := h.tester.Test(r.Context(), chi.URLParam(r, "group"), admin)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, res)
}
