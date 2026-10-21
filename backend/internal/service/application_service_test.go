package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type memApplications struct {
	created    []model.NewApplication
	recent     bool
	createErr  error
	idConflict int
}

func (m *memApplications) Create(_ context.Context, n model.NewApplication) error {
	if m.idConflict > 0 {
		m.idConflict--
		return &pgconn.PgError{Code: "23505", ConstraintName: "applicants_public_id_key"}
	}
	if m.createErr != nil {
		return m.createErr
	}
	m.created = append(m.created, n)
	return nil
}

func (m *memApplications) HasRecent(context.Context, model.NewApplication) (bool, error) {
	return m.recent, nil
}

type memApplicationFiles struct {
	mu       sync.Mutex
	uploaded map[uuid.UUID]UploadInput
	deleted  []uuid.UUID
	failName string
	maxBytes int64
}

func (m *memApplicationFiles) Upload(_ context.Context, in UploadInput) (model.File, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if in.Filename == m.failName {
		return model.File{}, apperr.New(422, "invalid_file_type", "The file content does not match its extension.")
	}
	id := uuid.New()
	m.uploaded[id] = in
	return model.File{ID: id}, nil
}

func (m *memApplicationFiles) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleted = append(m.deleted, id)
	return nil
}

func (m *memApplicationFiles) MaxUploadBytes(context.Context) (int64, error) { return m.maxBytes, nil }

type fakeGuard struct {
	err    error
	called int
	token  string
	ip     string
}

func (g *fakeGuard) Verify(_ context.Context, token, ip string) error {
	g.called++
	g.token, g.ip = token, ip
	return g.err
}

type sentEmail struct {
	code, to, relatedType string
	vars                  map[string]string
}

type fakeMailer struct {
	sent      []sentEmail
	notified  []sentEmail
	sendErr   error
	notifyErr error
}

func (m *fakeMailer) SendTemplate(_ context.Context, code, to string, vars map[string]string, relatedType string, _ *uuid.UUID) (model.EmailLog, error) {
	m.sent = append(m.sent, sentEmail{code: code, to: to, vars: vars, relatedType: relatedType})
	return model.EmailLog{}, m.sendErr
}

func (m *fakeMailer) Notify(_ context.Context, code string, vars map[string]string, relatedType string, _ *uuid.UUID) error {
	m.notified = append(m.notified, sentEmail{code: code, vars: vars, relatedType: relatedType})
	return m.notifyErr
}

type countingInvalidator struct{ calls int }

func (c *countingInvalidator) Invalidate(context.Context) error {
	c.calls++
	return nil
}

type applicationFixture struct {
	svc       *ApplicationService
	store     *memApplications
	vacancies *memVacancies
	files     *memApplicationFiles
	guard     *fakeGuard
	mailer    *fakeMailer
	dashboard *countingInvalidator
	rank      uuid.UUID
	vacancy   model.Vacancy
	now       time.Time
}

func newApplicationFixture(t *testing.T) *applicationFixture {
	t.Helper()
	f := &applicationFixture{
		store: &memApplications{}, vacancies: newMemVacancies(), files: &memApplicationFiles{uploaded: map[uuid.UUID]UploadInput{}, maxBytes: 5 << 20},
		guard: &fakeGuard{}, mailer: &fakeMailer{}, dashboard: &countingInvalidator{}, rank: uuid.New(),
		now: time.Date(2026, 10, 21, 3, 0, 0, 0, time.UTC),
	}
	f.vacancies.activeRanks[f.rank] = true
	f.vacancy = model.Vacancy{ID: uuid.New(), PublicID: "VAC222222", Slug: "chief-officer", Title: "Chief Officer", Status: model.VacancyOpen}
	f.vacancies.rows[f.vacancy.ID] = f.vacancy
	settings := staticSettings{"general": {"timezone": "Asia/Jakarta"}, "uploads": {"max_applicant_files": 4}}
	f.svc = NewApplicationService(f.store, f.vacancies, f.files, f.guard, f.mailer, f.dashboard, settings, "https://admin.example.com/", func() time.Time { return f.now })
	return f
}

func pdfPart(name string) ApplicationFile {
	return ApplicationFile{Filename: name, Size: 10, Open: func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("%PDF-1.7")), nil }}
}

func (f *applicationFixture) input() ApplicationInput {
	return ApplicationInput{
		Vacancy: "chief-officer", FullName: " Siti Aminah ", Email: "Siti@Example.com", Phone: "+6281234567890", DateOfBirth: "1990-05-01",
		Nationality: "Indonesian", RankAppliedID: f.rank.String(), YearsExperience: "8", Message: "Ready to join.", Consent: "true",
		TurnstileToken: "tok", IP: "203.0.113.9", CV: []ApplicationFile{pdfPart("cv.pdf")},
		Documents: []ApplicationFile{pdfPart("passport.pdf"), pdfPart("coc.pdf")},
	}
}

func appErrorFields(t *testing.T, err error, status int, code string) map[string]string {
	t.Helper()
	var e *apperr.Error
	require.ErrorAs(t, err, &e)
	assert.Equal(t, status, e.Status)
	assert.Equal(t, code, e.Code)
	return e.Fields
}

func requireAppError(t *testing.T, err error, status int, code string) {
	t.Helper()
	appErrorFields(t, err, status, code)
}

func TestSubmitApplicationStoresApplicantFilesAndQueuesEmails(t *testing.T) {
	f := newApplicationFixture(t)

	id, err := f.svc.Submit(context.Background(), f.input())

	require.NoError(t, err)
	require.Len(t, f.store.created, 1)
	n := f.store.created[0]
	assert.Equal(t, id, n.PublicID)
	assert.Len(t, id, 9)
	assert.Equal(t, f.vacancy.ID, n.VacancyID)
	assert.Equal(t, "Siti Aminah", n.FullName, "names are trimmed")
	assert.Equal(t, "siti@example.com", n.Email, "emails are lowercased")
	assert.Equal(t, f.rank, n.RankAppliedID)
	assert.Equal(t, 8, n.YearsExperience)
	assert.Equal(t, "203.0.113.9", n.IPAddress)
	assert.Equal(t, f.now, n.ConsentAt)
	assert.Equal(t, f.now.Add(-30*24*time.Hour), n.DuplicateSince)
	require.Len(t, n.Files, 3)
	assert.Equal(t, model.ApplicantFileCV, n.Files[0].Kind)
	assert.Equal(t, model.ApplicantFileDocument, n.Files[1].Kind)
	for _, file := range n.Files {
		in := f.files.uploaded[file.FileID]
		assert.Equal(t, "applicant", in.OwnerType)
		require.NotNil(t, in.OwnerID)
		assert.Equal(t, n.ID, *in.OwnerID, "files are owned by the applicant")
	}
	assert.Equal(t, "tok", f.guard.token)
	assert.Equal(t, "203.0.113.9", f.guard.ip)

	require.Len(t, f.mailer.sent, 1)
	assert.Equal(t, "applicant_confirmation", f.mailer.sent[0].code)
	assert.Equal(t, "siti@example.com", f.mailer.sent[0].to)
	assert.Equal(t, "applicant", f.mailer.sent[0].relatedType)
	assert.Equal(t, id, f.mailer.sent[0].vars["application_id"])
	assert.Equal(t, "Chief Officer", f.mailer.sent[0].vars["vacancy_title"])
	require.Len(t, f.mailer.notified, 1)
	assert.Equal(t, "admin_new_application", f.mailer.notified[0].code)
	assert.Equal(t, "https://admin.example.com/applicants/"+id, f.mailer.notified[0].vars["admin_url"])
	assert.Equal(t, 1, f.dashboard.calls)
}

func TestSubmitApplicationAcceptsVacancyPublicID(t *testing.T) {
	f := newApplicationFixture(t)
	in := f.input()
	in.Vacancy = "vac222222"

	_, err := f.svc.Submit(context.Background(), in)

	require.NoError(t, err)
	assert.Equal(t, f.vacancy.ID, f.store.created[0].VacancyID)
}

func TestSubmitApplicationDropsHoneypotSubmissionsSilently(t *testing.T) {
	f := newApplicationFixture(t)
	in := f.input()
	in.Website = "https://spam.example"

	id, err := f.svc.Submit(context.Background(), in)

	require.NoError(t, err)
	assert.Len(t, id, 9, "bots receive a plausible application ID")
	assert.Zero(t, f.guard.called, "the honeypot is checked before Turnstile")
	assert.Empty(t, f.store.created)
	assert.Empty(t, f.files.uploaded)
	assert.Empty(t, f.mailer.sent)
	assert.Empty(t, f.mailer.notified)
	assert.Zero(t, f.dashboard.calls)
}

func TestSubmitApplicationVerifiesTurnstileBeforeValidatingAnything(t *testing.T) {
	f := newApplicationFixture(t)
	f.guard.err = ErrTurnstileFailed
	in := ApplicationInput{TurnstileToken: "bad"}

	_, err := f.svc.Submit(context.Background(), in)

	requireAppError(t, err, 422, "turnstile_failed")
	assert.Empty(t, f.store.created)
	assert.Empty(t, f.files.uploaded)
}

func TestSubmitApplicationValidatesEveryField(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ApplicationInput)
		field  string
	}{
		{"missing vacancy", func(in *ApplicationInput) { in.Vacancy = "" }, "vacancy"},
		{"missing name", func(in *ApplicationInput) { in.FullName = "  " }, "full_name"},
		{"name too long", func(in *ApplicationInput) { in.FullName = strings.Repeat("a", 121) }, "full_name"},
		{"invalid email", func(in *ApplicationInput) { in.Email = "not-an-email" }, "email"},
		{"missing phone", func(in *ApplicationInput) { in.Phone = "" }, "phone"},
		{"phone not E.164", func(in *ApplicationInput) { in.Phone = "0812-345" }, "phone"},
		{"missing birth date", func(in *ApplicationInput) { in.DateOfBirth = "" }, "date_of_birth"},
		{"malformed birth date", func(in *ApplicationInput) { in.DateOfBirth = "01/05/1990" }, "date_of_birth"},
		{"younger than 16", func(in *ApplicationInput) { in.DateOfBirth = "2010-10-22" }, "date_of_birth"},
		{"older than 80", func(in *ApplicationInput) { in.DateOfBirth = "1945-10-20" }, "date_of_birth"},
		{"missing nationality", func(in *ApplicationInput) { in.Nationality = "" }, "nationality"},
		{"invalid rank id", func(in *ApplicationInput) { in.RankAppliedID = "master" }, "rank_applied_id"},
		{"missing experience", func(in *ApplicationInput) { in.YearsExperience = "" }, "years_experience"},
		{"experience not a number", func(in *ApplicationInput) { in.YearsExperience = "many" }, "years_experience"},
		{"experience above 60", func(in *ApplicationInput) { in.YearsExperience = "61" }, "years_experience"},
		{"negative experience", func(in *ApplicationInput) { in.YearsExperience = "-1" }, "years_experience"},
		{"message above 2000 characters", func(in *ApplicationInput) { in.Message = strings.Repeat("é", 2001) }, "message"},
		{"consent missing", func(in *ApplicationInput) { in.Consent = "" }, "consent"},
		{"consent false", func(in *ApplicationInput) { in.Consent = "false" }, "consent"},
		{"no CV", func(in *ApplicationInput) { in.CV = nil }, "cv"},
		{"two CVs", func(in *ApplicationInput) { in.CV = []ApplicationFile{pdfPart("a.pdf"), pdfPart("b.pdf")} }, "cv"},
		{"more supporting documents than allowed", func(in *ApplicationInput) {
			in.Documents = []ApplicationFile{pdfPart("1.pdf"), pdfPart("2.pdf"), pdfPart("3.pdf"), pdfPart("4.pdf"), pdfPart("5.pdf")}
		}, "documents"},
		{"oversized CV", func(in *ApplicationInput) { in.CV[0].Size = 6 << 20 }, "cv"},
		{"oversized document", func(in *ApplicationInput) { in.Documents[0].Size = 6 << 20 }, "documents"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newApplicationFixture(t)
			in := f.input()
			tc.mutate(&in)

			_, err := f.svc.Submit(context.Background(), in)

			fields := appErrorFields(t, err, 422, "validation_failed")
			assert.Contains(t, fields, tc.field)
			assert.Empty(t, f.store.created)
			assert.Empty(t, f.files.uploaded, "nothing is uploaded for an invalid form")
		})
	}
}

func TestSubmitApplicationAcceptsBoundaryAgesAndExperience(t *testing.T) {
	tests := []struct{ name, dob, years string }{
		{"exactly 16 today", "2010-10-21", "0"},
		{"exactly 80 today", "1946-10-21", "60"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newApplicationFixture(t)
			in := f.input()
			in.DateOfBirth, in.YearsExperience = tc.dob, tc.years

			_, err := f.svc.Submit(context.Background(), in)

			assert.NoError(t, err)
		})
	}
}

func TestSubmitApplicationRejectsVacanciesThatCannotTakeApplications(t *testing.T) {
	past := day(2026, 10, 20)
	tests := []struct {
		name   string
		mutate func(*model.Vacancy)
		slug   string
		status int
		code   string
	}{
		{"closed vacancy", func(v *model.Vacancy) { v.Status = model.VacancyClosed }, "chief-officer", 409, "vacancy_closed"},
		{"past closing date", func(v *model.Vacancy) { v.ClosingDate = past }, "chief-officer", 409, "vacancy_closed"},
		{"draft vacancy", func(v *model.Vacancy) { v.Status = model.VacancyDraft }, "chief-officer", 404, "not_found"},
		{"unknown vacancy", func(*model.Vacancy) {}, "nothing-here", 404, "not_found"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newApplicationFixture(t)
			v := f.vacancy
			tc.mutate(&v)
			f.vacancies.rows[v.ID] = v
			in := f.input()
			in.Vacancy = tc.slug

			_, err := f.svc.Submit(context.Background(), in)

			requireAppError(t, err, tc.status, tc.code)
			assert.Empty(t, f.files.uploaded)
			assert.Empty(t, f.store.created)
		})
	}
}

func TestSubmitApplicationAcceptsClosingDateToday(t *testing.T) {
	f := newApplicationFixture(t)
	v := f.vacancy
	v.ClosingDate = day(2026, 10, 21)
	f.vacancies.rows[v.ID] = v

	_, err := f.svc.Submit(context.Background(), f.input())

	assert.NoError(t, err)
}

func TestSubmitApplicationRequiresActiveRank(t *testing.T) {
	f := newApplicationFixture(t)
	f.vacancies.activeRanks[f.rank] = false

	_, err := f.svc.Submit(context.Background(), f.input())

	fields := appErrorFields(t, err, 422, "validation_failed")
	assert.Contains(t, fields, "rank_applied_id")
	assert.Empty(t, f.files.uploaded)
}

func TestSubmitApplicationRefusesDuplicateBeforeProcessingFiles(t *testing.T) {
	f := newApplicationFixture(t)
	f.store.recent = true

	_, err := f.svc.Submit(context.Background(), f.input())

	requireAppError(t, err, 409, "duplicate_application")
	assert.Empty(t, f.files.uploaded, "duplicates are refused before the expensive file pipeline")
	assert.Empty(t, f.mailer.sent)
}

func TestSubmitApplicationRemovesUploadedFilesWhenAnUploadFails(t *testing.T) {
	f := newApplicationFixture(t)
	f.files.failName = "coc.pdf"

	_, err := f.svc.Submit(context.Background(), f.input())

	fields := appErrorFields(t, err, 422, "invalid_file_type")
	assert.Contains(t, fields, "documents", "the error points at the failing form field")
	assert.Empty(t, f.store.created)
	assert.Len(t, f.files.deleted, len(f.files.uploaded), "every stored file is cleaned up")
	assert.Empty(t, f.mailer.sent)
}

func TestSubmitApplicationRemovesUploadedFilesWhenTheRaceForDuplicateIsLost(t *testing.T) {
	f := newApplicationFixture(t)
	f.store.createErr = repository.ErrDuplicateApplication

	_, err := f.svc.Submit(context.Background(), f.input())

	requireAppError(t, err, 409, "duplicate_application")
	assert.Len(t, f.files.uploaded, 3)
	assert.Len(t, f.files.deleted, 3)
	assert.Empty(t, f.mailer.sent)
}

func TestSubmitApplicationRemovesUploadedFilesWhenStoringFails(t *testing.T) {
	f := newApplicationFixture(t)
	f.store.createErr = errors.New("connection reset")

	_, err := f.svc.Submit(context.Background(), f.input())

	assert.Error(t, err)
	assert.Len(t, f.files.deleted, 3)
}

func TestSubmitApplicationRetriesOnPublicIDCollision(t *testing.T) {
	f := newApplicationFixture(t)
	f.store.idConflict = 2

	id, err := f.svc.Submit(context.Background(), f.input())

	require.NoError(t, err)
	assert.Equal(t, id, f.store.created[0].PublicID)
	assert.Empty(t, f.files.deleted)
}

func TestSubmitApplicationSucceedsWhenEmailsCannotBeQueued(t *testing.T) {
	f := newApplicationFixture(t)
	f.mailer.sendErr = errors.New("queue down")
	f.mailer.notifyErr = errors.New("queue down")

	id, err := f.svc.Submit(context.Background(), f.input())

	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Len(t, f.store.created, 1)
	assert.Empty(t, f.files.deleted, "stored files stay when only the emails failed")
}

func TestSubmitApplicationStripsControlCharactersFromText(t *testing.T) {
	f := newApplicationFixture(t)
	in := f.input()
	in.FullName = "Siti\x00 Aminah\r"
	in.Message = "Line one\nLine two\x07"

	_, err := f.svc.Submit(context.Background(), in)

	require.NoError(t, err)
	assert.Equal(t, "Siti Aminah", f.store.created[0].FullName)
	assert.Equal(t, "Line one\nLine two", f.store.created[0].Message)
}

func TestApplicationLimitsComeFromSettings(t *testing.T) {
	f := newApplicationFixture(t)

	limits, err := f.svc.Limits(context.Background())

	require.NoError(t, err)
	assert.Equal(t, ApplicationLimits{MaxDocuments: 4, MaxFileBytes: 5 << 20}, limits)
}

func TestSubmitAcceptsCVPlusTheMaximumNumberOfSupportingDocuments(t *testing.T) {
	f := newApplicationFixture(t)
	in := f.input()
	in.Documents = []ApplicationFile{pdfPart("1.pdf"), pdfPart("2.pdf"), pdfPart("3.pdf"), pdfPart("4.pdf")}

	_, err := f.svc.Submit(context.Background(), in)

	require.NoError(t, err)
	assert.Len(t, f.files.uploaded, 5)
}
