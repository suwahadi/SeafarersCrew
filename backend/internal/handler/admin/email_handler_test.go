package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/middleware"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type stubEmails struct {
	lastSend   service.EmailSendInput
	lastFilter model.EmailLogFilter
}

func (s *stubEmails) Send(_ context.Context, in service.EmailSendInput) (model.EmailLog, model.Job, error) {
	s.lastSend = in
	return model.EmailLog{ID: uuid.New(), To: in.To, Status: model.EmailQueued}, model.Job{ID: uuid.New()}, nil
}

func (s *stubEmails) Resend(context.Context, uuid.UUID) (model.EmailLog, model.Job, error) {
	return model.EmailLog{}, model.Job{}, apperr.Conflict("email_not_failed", "Only failed emails can be resent.")
}

func (s *stubEmails) ListLogs(_ context.Context, f model.EmailLogFilter, _, _ int) (repository.Page[model.EmailLog], error) {
	s.lastFilter = f
	return repository.Page[model.EmailLog]{Items: []model.EmailLog{{ID: uuid.New(), Status: model.EmailSent}}, Total: 1}, nil
}

func (s *stubEmails) Templates(context.Context) ([]model.EmailTemplate, error) {
	return []model.EmailTemplate{{Code: "send_cv", Name: "Send CV", BodyHTML: "<p>long</p>"}}, nil
}

func (s *stubEmails) Template(_ context.Context, code string) (model.EmailTemplate, error) {
	return model.EmailTemplate{Code: code, BodyHTML: "<p>x</p>"}, nil
}

func (s *stubEmails) UpdateTemplate(_ context.Context, code, subject, body string) (model.EmailTemplate, error) {
	return model.EmailTemplate{Code: code, Subject: subject, BodyHTML: body}, nil
}

func (s *stubEmails) PreviewTemplate(context.Context, string, string, string, map[string]string) (service.EmailPreview, error) {
	return service.EmailPreview{Subject: "s", BodyHTML: "b", UnknownTags: []string{}}, nil
}

type stubTester struct{ admin model.Admin }

func (s *stubTester) Test(_ context.Context, group string, admin model.Admin) (service.EmailConnectionResult, error) {
	s.admin = admin
	return service.EmailConnectionResult{OK: true, Message: group}, nil
}

func emailRouter(h *EmailHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/emails/send", h.Send)
	r.Get("/email-logs", h.Logs)
	r.Post("/email-logs/{id}/resend", h.Resend)
	r.Get("/email-templates", h.Templates)
	r.Get("/email-templates/{code}", h.Template)
	r.Post("/settings/{group}/test", h.TestConnection)
	return r
}

func TestSendEmailReturnsAccepted(t *testing.T) {
	emails := &stubEmails{}
	h := emailRouter(NewEmailHandler(emails, &stubTester{}))
	file := uuid.New()
	body := `{"template_code":"send_contract","to":["a@example.com"],"related_type":"contract","related_id":"ADFPKEX4K","attachment_file_ids":["` + file.String() + `"]}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(body)))
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	assert.Equal(t, "ADFPKEX4K", emails.lastSend.RelatedPublicID, "contracts are referenced by their public ID")
	assert.Nil(t, emails.lastSend.RelatedID)
	assert.Equal(t, []uuid.UUID{file}, emails.lastSend.AttachmentFileIDs)
	assert.Contains(t, rec.Body.String(), `"job_id"`)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(`{"to":["a@example.com"],"attachment_file_ids":["nope"]}`)))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestEmailLogsFiltersAndValidation(t *testing.T) {
	emails := &stubEmails{}
	h := emailRouter(NewEmailHandler(emails, &stubTester{}))
	id := uuid.New()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/email-logs?related_type=crew&related_id="+id.String()+"&status=failed", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, model.EmailLogFilter{RelatedType: "crew", RelatedID: &id, Status: "failed"}, emails.lastFilter)
	var body struct {
		Meta struct{ Total int } `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 1, body.Meta.Total)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/email-logs?status=bounced", nil))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestResendErrors(t *testing.T) {
	h := emailRouter(NewEmailHandler(&stubEmails{}, &stubTester{}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/email-logs/not-a-uuid/resend", nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/email-logs/"+uuid.NewString()+"/resend", nil))
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestTemplateListOmitsBodiesAndDetailListsTags(t *testing.T) {
	h := emailRouter(NewEmailHandler(&stubEmails{}, &stubTester{}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/email-templates", nil))
	assert.NotContains(t, rec.Body.String(), "long")

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/email-templates/send_cv", nil))
	assert.Contains(t, rec.Body.String(), `"available_tags":["crew_name"`)
}

func TestTestConnectionPassesSignedInAdmin(t *testing.T) {
	tester := &stubTester{}
	h := emailRouter(NewEmailHandler(&stubEmails{}, tester))
	req := httptest.NewRequest(http.MethodPost, "/settings/email/test", nil)
	req = req.WithContext(middleware.WithAdmin(req.Context(), model.Admin{Email: "me@example.com"}, "sid"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "me@example.com", tester.admin.Email)
	assert.JSONEq(t, `{"data":{"ok":true,"message":"email"}}`, rec.Body.String())
}
