package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/job"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/brevo"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/turnstile"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type memEmailStore struct {
	templates map[string]model.EmailTemplate
	logs      map[uuid.UUID]model.EmailLog
	payloads  map[uuid.UUID]json.RawMessage
}

func newMemEmailStore() *memEmailStore {
	return &memEmailStore{
		templates: map[string]model.EmailTemplate{
			"send_contract": {Code: "send_contract", Name: "Send Contract", Subject: "Contract {contract_id} – {crew_name}",
				BodyHTML: "<p>Dear {crew_name},</p><p>{company_name}</p><p>{mystery}</p>"},
		},
		logs:     map[uuid.UUID]model.EmailLog{},
		payloads: map[uuid.UUID]json.RawMessage{},
	}
}

func (m *memEmailStore) ListTemplates(context.Context) ([]model.EmailTemplate, error) {
	var out []model.EmailTemplate
	for _, t := range m.templates {
		out = append(out, t)
	}
	return out, nil
}

func (m *memEmailStore) GetTemplate(_ context.Context, code string) (model.EmailTemplate, error) {
	t, ok := m.templates[code]
	if !ok {
		return t, repository.ErrNotFound
	}
	return t, nil
}

func (m *memEmailStore) UpdateTemplate(_ context.Context, code, subject, body string) (model.EmailTemplate, error) {
	t := m.templates[code]
	t.Subject, t.BodyHTML = subject, body
	m.templates[code] = t
	return t, nil
}

func (m *memEmailStore) CreateLog(_ context.Context, l model.EmailLog) (model.EmailLog, error) {
	l.ID, l.Status = uuid.New(), model.EmailQueued
	m.logs[l.ID] = l
	return l, nil
}

func (m *memEmailStore) GetLog(_ context.Context, id uuid.UUID) (model.EmailLog, error) {
	l, ok := m.logs[id]
	if !ok {
		return l, repository.ErrNotFound
	}
	return l, nil
}

func (m *memEmailStore) ListLogs(context.Context, model.EmailLogFilter, int, int) (repository.Page[model.EmailLog], error) {
	return repository.Page[model.EmailLog]{}, nil
}

func (m *memEmailStore) update(id uuid.UUID, fn func(*model.EmailLog)) error {
	l, ok := m.logs[id]
	if !ok {
		return repository.ErrNotFound
	}
	fn(&l)
	l.Attempts++
	m.logs[id] = l
	return nil
}

func (m *memEmailStore) MarkSent(_ context.Context, id uuid.UUID, msgID string, at time.Time) error {
	return m.update(id, func(l *model.EmailLog) {
		l.Status, l.BrevoMessageID, l.SentAt, l.Error = model.EmailSent, msgID, &at, ""
	})
}

func (m *memEmailStore) MarkFailed(_ context.Context, id uuid.UUID, msg string) error {
	return m.update(id, func(l *model.EmailLog) { l.Status, l.Error = model.EmailFailed, msg })
}

func (m *memEmailStore) MarkRetrying(_ context.Context, id uuid.UUID, msg string) error {
	return m.update(id, func(l *model.EmailLog) { l.Status, l.Error = model.EmailQueued, msg })
}

func (m *memEmailStore) RequeueFailed(_ context.Context, id uuid.UUID) (bool, error) {
	l := m.logs[id]
	if l.Status != model.EmailFailed {
		return false, nil
	}
	l.Status, l.Error = model.EmailQueued, ""
	m.logs[id] = l
	return true, nil
}

func (m *memEmailStore) LatestJobPayload(_ context.Context, _ string, id uuid.UUID) (json.RawMessage, error) {
	p, ok := m.payloads[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return p, nil
}

type emailQueue struct {
	store *memEmailStore
	jobs  []model.Job
}

func (q *emailQueue) Enqueue(_ context.Context, kind string, payload any) (model.Job, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return model.Job{}, err
	}
	j := model.Job{ID: uuid.New(), Kind: kind, Status: model.JobQueued, Payload: raw}
	var p emailJobPayload
	if json.Unmarshal(raw, &p) == nil && q.store != nil {
		q.store.payloads[p.LogID] = raw
	}
	q.jobs = append(q.jobs, j)
	return j, nil
}

type fakeDelivery struct {
	sent    []brevo.Message
	err     error
	account string
}

func (f *fakeDelivery) Send(_ context.Context, m brevo.Message) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.sent = append(f.sent, m)
	return "<msg-1@brevo>", nil
}

func (f *fakeDelivery) Account(context.Context, string) (string, error) { return f.account, f.err }

type fakeAttachmentLoader map[uuid.UUID]brevo.Attachment

func (f fakeAttachmentLoader) Load(_ context.Context, id uuid.UUID) (brevo.Attachment, error) {
	a, ok := f[id]
	if !ok {
		return a, repository.ErrNotFound
	}
	return a, nil
}

type emailFixture struct {
	svc      *EmailService
	store    *memEmailStore
	queue    *emailQueue
	delivery *fakeDelivery
	files    fakeAttachmentLoader
	settings staticSettings
}

func newEmailFixture() emailFixture {
	store := newMemEmailStore()
	f := emailFixture{
		store:    store,
		queue:    &emailQueue{store: store},
		delivery: &fakeDelivery{},
		files:    fakeAttachmentLoader{},
		settings: staticSettings{
			"email":   {"brevo_api_key": "xkeysib-1", "sender_email": "no-reply@example.com", "sender_name": "Seafarers Crew", "reply_to": "ops@example.com", "admin_notification_emails": []string{"admin@example.com", "ops@example.com"}},
			"company": {"legal_name": "PT Ocean Crew", "phone": "+62210000000"},
			"general": {"timezone": "Asia/Jakarta", "date_format": "DD-MMM-YYYY"},
			"alerts":  {},
		},
	}
	now := func() time.Time { return time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC) }
	f.svc = NewEmailService(store, f.delivery, f.files, f.settings, f.queue, fakeRelated{}, now)
	return f
}

func (f emailFixture) runLastJob(t *testing.T, attempt int) error {
	t.Helper()
	j := f.queue.jobs[len(f.queue.jobs)-1]
	j.Attempts = attempt
	_, err := f.svc.Handle(context.Background(), j)
	return err
}

func TestRenderEmailEscapesValuesAndDropsUnknownTags(t *testing.T) {
	got := RenderEmail("<p>Hi {name}, {unknown}{ name }</p>", map[string]string{"name": `<script>"x"</script>`}, true)
	assert.Equal(t, "<p>Hi &lt;script&gt;&#34;x&#34;&lt;/script&gt;, { name }</p>", got)
	assert.Equal(t, "Hi <b>", RenderEmail("Hi {name}", map[string]string{"name": "<b>"}, false))
	assert.Equal(t, "Line one two", RenderEmailSubject("Line {a}", map[string]string{"a": "one\r\ntwo"}))
	assert.Equal(t, []string{"a", "b"}, EmailTags("{a} {b} {a} {Bad}"))
}

func TestEmailAvailableTagsIncludeCompanyTags(t *testing.T) {
	tags := EmailAvailableTags("send_cv")
	assert.Contains(t, tags, "crew_name")
	assert.Contains(t, tags, "company_name")
	assert.Contains(t, tags, "today")
}

func TestSendValidatesRecipients(t *testing.T) {
	f := newEmailFixture()
	many := make([]string, 21)
	for i := range many {
		many[i] = strings.Repeat("a", i+1) + "@example.com"
	}
	cases := []struct {
		name  string
		in    EmailSendInput
		field string
	}{
		{"no recipient", EmailSendInput{TemplateCode: "send_contract"}, "to"},
		{"bad recipient", EmailSendInput{TemplateCode: "send_contract", To: []string{"not-an-email"}}, "to"},
		{"display name not allowed", EmailSendInput{TemplateCode: "send_contract", To: []string{"Budi <budi@example.com>"}}, "to"},
		{"bad cc", EmailSendInput{TemplateCode: "send_contract", To: []string{"a@example.com"}, CC: []string{"x@"}}, "cc"},
		{"too many", EmailSendInput{TemplateCode: "send_contract", To: many}, "to"},
		{"unknown related type", EmailSendInput{TemplateCode: "send_contract", To: []string{"a@example.com"}, RelatedType: "ship", RelatedID: &uuid.UUID{}}, "related_type"},
		{"related id without type", EmailSendInput{TemplateCode: "send_contract", To: []string{"a@example.com"}, RelatedID: &uuid.UUID{}}, "related_id"},
		{"no template no body", EmailSendInput{To: []string{"a@example.com"}}, "subject"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := f.svc.Send(context.Background(), tc.in)
			e := requireAppErr(t, err, "validation_failed")
			assert.Contains(t, e.Fields, tc.field)
		})
	}
	_, _, err := f.svc.Send(context.Background(), EmailSendInput{TemplateCode: "missing", To: []string{"a@example.com"}})
	requireAppErr(t, err, "not_found")
}

func TestSendRendersTemplateLogsAndQueues(t *testing.T) {
	f := newEmailFixture()
	contract := uuid.New()
	log, j, err := f.svc.Send(context.Background(), EmailSendInput{
		TemplateCode: "send_contract", To: []string{" Budi@Example.com ", "budi@example.com"}, CC: []string{"ops@example.com"},
		Variables: map[string]string{"crew_name": "Budi <Santoso>", "contract_id": "CT4K9P2XW"}, RelatedType: "contract", RelatedID: &contract,
	})
	require.NoError(t, err)
	assert.Equal(t, EmailSendJob, j.Kind)
	assert.Equal(t, []string{"budi@example.com"}, log.To, "addresses are normalised and deduplicated")
	assert.Equal(t, "Contract CT4K9P2XW – Budi <Santoso>", log.Subject)
	assert.Equal(t, model.EmailQueued, log.Status)

	var p emailJobPayload
	require.NoError(t, json.Unmarshal(j.Payload, &p))
	assert.Equal(t, log.ID, p.LogID)
	assert.Contains(t, p.BodyHTML, "Dear Budi &lt;Santoso&gt;")
	assert.Contains(t, p.BodyHTML, "PT Ocean Crew", "company tags come from Settings")
	assert.NotContains(t, p.BodyHTML, "{mystery}")
}

func TestSendSanitizesCustomBody(t *testing.T) {
	f := newEmailFixture()
	_, j, err := f.svc.Send(context.Background(), EmailSendInput{To: []string{"a@example.com"}, Subject: "Hi", BodyHTML: `<p onclick="x()">Hi</p><script>alert(1)</script>`})
	require.NoError(t, err)
	var p emailJobPayload
	require.NoError(t, json.Unmarshal(j.Payload, &p))
	assert.Equal(t, "<p>Hi</p>", p.BodyHTML)
}

func TestHandleSendsWithSettingsAndAttachments(t *testing.T) {
	f := newEmailFixture()
	fileID := uuid.New()
	f.files[fileID] = brevo.Attachment{Name: "sea.pdf", Content: []byte("%PDF")}
	log, _, err := f.svc.Send(context.Background(), EmailSendInput{TemplateCode: "send_contract", To: []string{"a@example.com"}, AttachmentFileIDs: []uuid.UUID{fileID}})
	require.NoError(t, err)

	require.NoError(t, f.runLastJob(t, 1))
	require.Len(t, f.delivery.sent, 1)
	m := f.delivery.sent[0]
	assert.Equal(t, "xkeysib-1", m.APIKey)
	assert.Equal(t, brevo.Address{Email: "no-reply@example.com", Name: "Seafarers Crew"}, m.Sender)
	assert.Equal(t, "ops@example.com", m.ReplyTo.Email)
	assert.Equal(t, "sea.pdf", m.Attachments[0].Name)

	saved := f.store.logs[log.ID]
	assert.Equal(t, model.EmailSent, saved.Status)
	assert.Equal(t, "<msg-1@brevo>", saved.BrevoMessageID)
	assert.Equal(t, 1, saved.Attempts)

	require.NoError(t, f.runLastJob(t, 2), "an already sent log is not sent twice")
	assert.Len(t, f.delivery.sent, 1)
}

func TestHandleRetriesTransientErrorsThenFails(t *testing.T) {
	f := newEmailFixture()
	log, _, err := f.svc.Send(context.Background(), EmailSendInput{TemplateCode: "send_contract", To: []string{"a@example.com"}})
	require.NoError(t, err)
	f.delivery.err = &brevo.Error{Status: 503, Message: "unavailable", Retryable: true}

	err = f.runLastJob(t, 1)
	require.Error(t, err)
	var perm job.Permanent
	assert.False(t, errors.As(err, &perm), "a transient failure is retried by the queue")
	assert.Equal(t, model.EmailQueued, f.store.logs[log.ID].Status)

	err = f.runLastJob(t, 3)
	assert.True(t, errors.As(err, &perm), "the final attempt is permanent")
	saved := f.store.logs[log.ID]
	assert.Equal(t, model.EmailFailed, saved.Status)
	assert.Contains(t, saved.Error, "unavailable")
	assert.Equal(t, 2, saved.Attempts)
}

func TestHandleFailsImmediatelyOnPermanentErrors(t *testing.T) {
	cases := map[string]func(f *emailFixture) []uuid.UUID{
		"client error": func(f *emailFixture) []uuid.UUID {
			f.delivery.err = &brevo.Error{Status: 400, Message: "invalid email"}
			return nil
		},
		"not configured": func(f *emailFixture) []uuid.UUID {
			f.settings["email"] = map[string]any{}
			return nil
		},
		"attachments too large": func(f *emailFixture) []uuid.UUID {
			a, b := uuid.New(), uuid.New()
			f.files[a] = brevo.Attachment{Name: "a.pdf", Content: make([]byte, 6<<20)}
			f.files[b] = brevo.Attachment{Name: "b.pdf", Content: make([]byte, 5<<20)}
			return []uuid.UUID{a, b}
		},
		"attachment deleted": func(*emailFixture) []uuid.UUID { return []uuid.UUID{uuid.New()} },
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			f := newEmailFixture()
			files := setup(&f)
			log, _, err := f.svc.Send(context.Background(), EmailSendInput{TemplateCode: "send_contract", To: []string{"a@example.com"}, AttachmentFileIDs: files})
			require.NoError(t, err)
			err = f.runLastJob(t, 1)
			var perm job.Permanent
			assert.True(t, errors.As(err, &perm))
			assert.Equal(t, model.EmailFailed, f.store.logs[log.ID].Status)
			assert.Empty(t, f.delivery.sent)
		})
	}
}

func TestResendRequeuesOnlyFailedLogsWithOriginalBody(t *testing.T) {
	f := newEmailFixture()
	log, first, err := f.svc.Send(context.Background(), EmailSendInput{TemplateCode: "send_contract", To: []string{"a@example.com"}, Variables: map[string]string{"crew_name": "Budi"}})
	require.NoError(t, err)

	_, _, err = f.svc.Resend(context.Background(), log.ID)
	requireAppErr(t, err, "email_not_failed")

	f.delivery.err = &brevo.Error{Status: 401, Message: "unauthorized"}
	require.Error(t, f.runLastJob(t, 1))
	f.delivery.err = nil

	resent, second, err := f.svc.Resend(context.Background(), log.ID)
	require.NoError(t, err)
	assert.Equal(t, model.EmailQueued, resent.Status)
	assert.JSONEq(t, string(first.Payload), string(second.Payload))
	require.NoError(t, f.runLastJob(t, 1))
	assert.Equal(t, model.EmailSent, f.store.logs[log.ID].Status)
	assert.Contains(t, f.delivery.sent[0].HTML, "Dear Budi")

	_, _, err = f.svc.Resend(context.Background(), uuid.New())
	requireAppErr(t, err, "not_found")
}

func TestNotifySendsToAdminAddressesOrSkips(t *testing.T) {
	f := newEmailFixture()
	require.NoError(t, f.svc.Notify(context.Background(), "send_contract", nil, "", nil))
	require.Len(t, f.queue.jobs, 1)
	var p emailJobPayload
	require.NoError(t, json.Unmarshal(f.queue.jobs[0].Payload, &p))
	assert.Equal(t, []string{"admin@example.com", "ops@example.com"}, f.store.logs[p.LogID].To)

	f.settings["email"]["admin_notification_emails"] = []string{}
	require.NoError(t, f.svc.Notify(context.Background(), "send_contract", nil, "", nil))
	assert.Len(t, f.queue.jobs, 1)
}

func TestSendTemplateQueuesForOneRecipient(t *testing.T) {
	f := newEmailFixture()
	id := uuid.New()
	log, err := f.svc.SendTemplate(context.Background(), "send_contract", "siti@example.com", map[string]string{"crew_name": "Siti"}, "applicant", &id)
	require.NoError(t, err)
	assert.Equal(t, []string{"siti@example.com"}, log.To)
	assert.Equal(t, "applicant", log.RelatedType)
}

func TestUpdateTemplateValidatesAndSanitizes(t *testing.T) {
	f := newEmailFixture()
	_, err := f.svc.UpdateTemplate(context.Background(), "send_contract", "", "")
	e := requireAppErr(t, err, "validation_failed")
	assert.Contains(t, e.Fields, "subject")
	assert.Contains(t, e.Fields, "body_html")

	got, err := f.svc.UpdateTemplate(context.Background(), "send_contract", "New\nsubject", `<p>Hi {crew_name}</p><img src=x onerror="alert(1)">`)
	require.NoError(t, err)
	assert.Equal(t, "New subject", got.Subject)
	assert.NotContains(t, got.BodyHTML, "onerror")
	assert.Contains(t, got.BodyHTML, "{crew_name}")

	_, err = f.svc.UpdateTemplate(context.Background(), "nope", "a", "b")
	requireAppErr(t, err, "not_found")
}

func TestPreviewTemplateUsesSamplesAndFlagsUnknownTags(t *testing.T) {
	f := newEmailFixture()
	p, err := f.svc.PreviewTemplate(context.Background(), "send_contract", "", "", map[string]string{"crew_name": "Override"})
	require.NoError(t, err)
	assert.Equal(t, "Contract CT4K9P2XW – Override", p.Subject)
	assert.Contains(t, p.BodyHTML, "PT Ocean Crew")
	assert.Equal(t, []string{"mystery"}, p.UnknownTags)

	p, err = f.svc.PreviewTemplate(context.Background(), "send_contract", "Draft {today}", "<p>x</p>", nil)
	require.NoError(t, err)
	assert.Equal(t, "Draft 20-Oct-2026", p.Subject)
	assert.Empty(t, p.UnknownTags)
}

type fakeStorageTester struct{ err error }

func (f fakeStorageTester) Test(context.Context) error { return f.err }

type fakeVerifier struct {
	res turnstile.Result
	err error
}

func (f fakeVerifier) Verify(context.Context, string, string, string) (turnstile.Result, error) {
	return f.res, f.err
}

func TestIntegrationTester(t *testing.T) {
	admin := model.Admin{Email: "me@example.com"}
	settings := staticSettings{
		"email":    {"brevo_api_key": "k", "sender_email": "no-reply@example.com"},
		"security": {"turnstile_site_key": "site", "turnstile_secret_key": "secret"},
	}
	ctx := context.Background()

	delivery := &fakeDelivery{}
	tester := NewEmailIntegrationTester(delivery, fakeStorageTester{}, fakeVerifier{res: turnstile.Result{ErrorCodes: []string{"invalid-input-response"}}}, settings)
	res, err := tester.Test(ctx, "email", admin)
	require.NoError(t, err)
	assert.True(t, res.OK)
	assert.Equal(t, "me@example.com", delivery.sent[0].To[0].Email)

	res, err = tester.Test(ctx, "storage", admin)
	require.NoError(t, err)
	assert.True(t, res.OK)

	res, err = tester.Test(ctx, "security", admin)
	require.NoError(t, err)
	assert.True(t, res.OK, "a rejected dummy token with a valid secret means the secret works")

	failing := NewEmailIntegrationTester(&fakeDelivery{err: &brevo.Error{Status: 401, Message: "key not found"}}, fakeStorageTester{err: errors.New("access denied")},
		fakeVerifier{res: turnstile.Result{ErrorCodes: []string{"invalid-input-secret"}}}, settings)
	for _, group := range []string{"email", "storage", "security"} {
		res, err := failing.Test(ctx, group, admin)
		require.NoError(t, err)
		assert.False(t, res.OK, group)
		assert.NotEmpty(t, res.Message)
	}

	unconfigured := NewEmailIntegrationTester(delivery, fakeStorageTester{}, fakeVerifier{}, staticSettings{"email": {}, "security": {}})
	res, err = unconfigured.Test(ctx, "email", admin)
	require.NoError(t, err)
	assert.False(t, res.OK)
	res, err = unconfigured.Test(ctx, "security", admin)
	require.NoError(t, err)
	assert.False(t, res.OK)

	_, err = tester.Test(ctx, "branding", admin)
	requireAppErr(t, err, "not_found")
}

type fakeRelated struct{}

var relatedContractID = uuid.MustParse("01a15ab4-0000-7000-8000-000000000001")

func (fakeRelated) EmailContext(_ context.Context, relatedType, publicID string) (uuid.UUID, map[string]string, error) {
	if relatedType != "contract" || publicID != "ADFPKEX4K" {
		return uuid.Nil, nil, apperr.NotFound("Contract")
	}
	return relatedContractID, map[string]string{"crew_name": "Budi Santoso", "contract_id": "ADFPKEX4K", "rank": "Master"}, nil
}

func TestSendResolvesPublicIDAndFillsTemplateVariables(t *testing.T) {
	f := newEmailFixture()
	log, _, err := f.svc.Send(context.Background(), EmailSendInput{
		TemplateCode: "send_contract", To: []string{"capt@example.com"}, RelatedType: "contract", RelatedPublicID: "ADFPKEX4K",
		Variables: map[string]string{"crew_name": "Capt. Budi"},
	})
	require.NoError(t, err)
	assert.Equal(t, relatedContractID, *log.RelatedID)
	assert.Equal(t, "Contract ADFPKEX4K – Capt. Budi", log.Subject, "record values fill tags and explicit variables override them")

	_, _, err = f.svc.Send(context.Background(), EmailSendInput{TemplateCode: "send_contract", To: []string{"capt@example.com"}, RelatedType: "contract", RelatedPublicID: "NOPE23456"})
	requireAppErr(t, err, "not_found")
}
