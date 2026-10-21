package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/filecheck"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/logctx"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/publicid"
)

// Application limits from PRD 8.2.
const (
	ApplicationDuplicateWindow = 30 * 24 * time.Hour
	ApplicantMinAge            = 16
	ApplicantMaxAge            = 80
	ApplicationMaxMessage      = 2000
	applicationUploadWorkers   = 3
)

// ApplicationStore persists public applications.
type ApplicationStore interface {
	Create(ctx context.Context, n model.NewApplication) error
	HasRecent(ctx context.Context, n model.NewApplication) (bool, error)
}

// ApplicationVacancies finds the vacancy an application targets.
type ApplicationVacancies interface {
	GetByPublicID(ctx context.Context, publicID string) (model.Vacancy, error)
	GetBySlug(ctx context.Context, slug string) (model.Vacancy, error)
	RankActive(ctx context.Context, id uuid.UUID) (bool, error)
}

// ApplicationFiles runs the upload pipeline for applicant files.
type ApplicationFiles interface {
	Upload(ctx context.Context, in UploadInput) (model.File, error)
	Delete(ctx context.Context, id uuid.UUID) error
	MaxUploadBytes(ctx context.Context) (int64, error)
}

// SubmissionGuard verifies a public form submission before it is processed.
type SubmissionGuard interface {
	Verify(ctx context.Context, token, ip string) error
}

// SubmissionMailer queues emails about public submissions.
type SubmissionMailer interface {
	SendTemplate(ctx context.Context, code, to string, vars map[string]string, relatedType string, relatedID *uuid.UUID) (model.EmailLog, error)
	Notify(ctx context.Context, code string, vars map[string]string, relatedType string, relatedID *uuid.UUID) error
}

// CacheInvalidator drops cached aggregates such as the dashboard counters.
type CacheInvalidator interface {
	Invalidate(ctx context.Context) error
}

// ApplicationFile is one uploaded file part of an application.
type ApplicationFile struct {
	Filename string
	Size     int64
	Open     func() (io.ReadCloser, error)
}

// ApplicationInput is the raw content of the public application form.
type ApplicationInput struct {
	Vacancy         string
	FullName        string
	Email           string
	Phone           string
	DateOfBirth     string
	Nationality     string
	RankAppliedID   string
	YearsExperience string
	Message         string
	Consent         string
	Website         string
	TurnstileToken  string
	IP              string
	CV              []ApplicationFile
	Documents       []ApplicationFile
}

// ApplicationLimits are the upload limits shown to applicants.
type ApplicationLimits struct {
	MaxDocuments int
	MaxFileBytes int64
}

// ApplicationService receives public job applications (PRD 8.2, 8.4).
type ApplicationService struct {
	store      ApplicationStore
	vacancies  ApplicationVacancies
	files      ApplicationFiles
	guard      SubmissionGuard
	mailer     SubmissionMailer
	dashboard  CacheInvalidator
	settings   SettingsReader
	adminURL   string
	now        func() time.Time
	newID      func() (string, error)
	newOwnerID func() (uuid.UUID, error)
}

// NewApplicationService builds an ApplicationService.
func NewApplicationService(store ApplicationStore, vacancies ApplicationVacancies, files ApplicationFiles, guard SubmissionGuard,
	mailer SubmissionMailer, dashboard CacheInvalidator, settings SettingsReader, adminURL string, now func() time.Time) *ApplicationService {
	return &ApplicationService{store: store, vacancies: vacancies, files: files, guard: guard, mailer: mailer, dashboard: dashboard,
		settings: settings, adminURL: strings.TrimRight(adminURL, "/"), now: now, newID: publicid.New, newOwnerID: uuid.NewV7}
}

// Limits returns the file count and size limits of an application.
func (s *ApplicationService) Limits(ctx context.Context) (ApplicationLimits, error) {
	maxBytes, err := s.files.MaxUploadBytes(ctx)
	if err != nil {
		return ApplicationLimits{}, err
	}
	values, err := s.settings.Values(ctx, "uploads")
	if err != nil {
		return ApplicationLimits{}, err
	}
	maxFiles := uploadInt(values["max_applicant_files"], 10)
	return ApplicationLimits{MaxDocuments: maxFiles, MaxFileBytes: maxBytes}, nil
}

// Submit processes an application and returns its Application ID. A filled honeypot field is answered with a
// plausible ID and nothing is stored or emailed.
func (s *ApplicationService) Submit(ctx context.Context, in ApplicationInput) (string, error) {
	if strings.TrimSpace(in.Website) != "" {
		return s.newID()
	}
	if err := s.guard.Verify(ctx, in.TurnstileToken, in.IP); err != nil {
		return "", err
	}
	limits, err := s.Limits(ctx)
	if err != nil {
		return "", err
	}
	today, err := s.today(ctx)
	if err != nil {
		return "", err
	}
	n, err := s.validate(in, limits, today)
	if err != nil {
		return "", err
	}
	vacancy, err := s.openVacancy(ctx, in.Vacancy, today)
	if err != nil {
		return "", err
	}
	if err := s.checkRank(ctx, n.RankAppliedID); err != nil {
		return "", err
	}
	n.VacancyID, n.ConsentAt, n.DuplicateSince = vacancy.ID, s.now().UTC(), s.now().UTC().Add(-ApplicationDuplicateWindow)
	if err := s.checkDuplicate(ctx, n); err != nil {
		return "", err
	}
	if n.ID, err = s.newOwnerID(); err != nil {
		return "", fmt.Errorf("generate applicant id: %w", err)
	}
	if n.Files, err = s.storeFiles(ctx, n.ID, in, limits); err != nil {
		return "", err
	}
	publicID, err := s.insert(ctx, n)
	if err != nil {
		s.discard(ctx, n.Files)
		return "", err
	}
	s.announce(ctx, n, publicID, vacancy)
	return publicID, nil
}

func (s *ApplicationService) today(ctx context.Context) (time.Time, error) {
	loc, _, err := LoadDisplaySettings(ctx, s.settings)
	if err != nil {
		return time.Time{}, err
	}
	return datetime.Today(s.now(), loc), nil
}

func (s *ApplicationService) validate(in ApplicationInput, limits ApplicationLimits, today time.Time) (model.NewApplication, error) {
	e := request.Errors{}
	n := model.NewApplication{
		FullName:    plainText(in.FullName, false),
		Email:       strings.ToLower(strings.TrimSpace(in.Email)),
		Phone:       strings.TrimSpace(in.Phone),
		Nationality: plainText(in.Nationality, false),
		Message:     plainText(in.Message, true),
		IPAddress:   cleanIP(in.IP),
	}
	if strings.TrimSpace(in.Vacancy) == "" {
		e["vacancy"] = "This field is required."
	}
	e.Required("full_name", n.FullName, 120)
	e.Required("email", n.Email, 254)
	e.Email("email", n.Email)
	e.Required("phone", n.Phone, 20)
	e.Phone("phone", n.Phone)
	e.Required("nationality", n.Nationality, 80)
	e.MaxLen("message", n.Message, ApplicationMaxMessage)
	n.RankAppliedID = e.RequiredUUID("rank_applied_id", strings.TrimSpace(in.RankAppliedID))
	n.DateOfBirth = s.validateBirthDate(e, strings.TrimSpace(in.DateOfBirth), today)
	n.YearsExperience = validateYears(e, strings.TrimSpace(in.YearsExperience))
	if !truthy(in.Consent) {
		e["consent"] = "You must accept the privacy policy to apply."
	}
	validateApplicationFiles(e, in, limits)
	if len(e) > 0 {
		return model.NewApplication{}, apperr.Validation(e)
	}
	return n, nil
}

func (s *ApplicationService) validateBirthDate(e request.Errors, raw string, today time.Time) time.Time {
	dob := e.RequiredDate("date_of_birth", raw)
	if dob.IsZero() {
		return dob
	}
	if age := datetime.Age(&dob, today); age != nil && (*age < ApplicantMinAge || *age > ApplicantMaxAge) {
		e["date_of_birth"] = fmt.Sprintf("Applicants must be between %d and %d years old.", ApplicantMinAge, ApplicantMaxAge)
	}
	return dob
}

func validateYears(e request.Errors, raw string) int {
	if raw == "" {
		e["years_experience"] = "This field is required."
		return 0
	}
	years, err := strconv.Atoi(raw)
	if err != nil {
		e["years_experience"] = "This must be a whole number."
		return 0
	}
	e.Range("years_experience", &years, 0, 60)
	return years
}

func validateApplicationFiles(e request.Errors, in ApplicationInput, limits ApplicationLimits) {
	switch {
	case len(in.CV) == 0:
		e["cv"] = "A CV is required."
	case len(in.CV) > 1:
		e["cv"] = "Upload exactly one CV."
	}
	if len(in.Documents) > limits.MaxDocuments {
		e["documents"] = fmt.Sprintf("You can upload at most %d supporting documents.", limits.MaxDocuments)
	}
	tooLarge := fmt.Sprintf("Each file may not be larger than %d MB.", limits.MaxFileBytes>>20)
	for _, f := range in.CV {
		if f.Size > limits.MaxFileBytes {
			e["cv"] = tooLarge
		}
	}
	for _, f := range in.Documents {
		if f.Size > limits.MaxFileBytes {
			e["documents"] = tooLarge
		}
	}
}

func (s *ApplicationService) openVacancy(ctx context.Context, ref string, today time.Time) (model.Vacancy, error) {
	ref = strings.TrimSpace(ref)
	var v model.Vacancy
	var err error
	if upper := strings.ToUpper(ref); publicid.Valid(upper) {
		v, err = s.vacancies.GetByPublicID(ctx, upper)
	} else {
		err = repository.ErrNotFound
	}
	if errors.Is(err, repository.ErrNotFound) {
		v, err = s.vacancies.GetBySlug(ctx, ref)
	}
	if errors.Is(err, repository.ErrNotFound) || (err == nil && v.Status == model.VacancyDraft) {
		return model.Vacancy{}, apperr.NotFound("Vacancy")
	}
	if err != nil {
		return model.Vacancy{}, err
	}
	if !VacancyAcceptsApplications(v, today) {
		return model.Vacancy{}, apperr.Conflict("vacancy_closed", "This vacancy is no longer accepting applications.")
	}
	return v, nil
}

func (s *ApplicationService) checkRank(ctx context.Context, rankID uuid.UUID) error {
	ok, err := s.vacancies.RankActive(ctx, rankID)
	if err != nil {
		return err
	}
	if !ok {
		return apperr.Validation(map[string]string{"rank_applied_id": "Choose a rank from the list."})
	}
	return nil
}

var errDuplicateApplication = apperr.Conflict("duplicate_application", "You have already applied for this vacancy in the last 30 days.")

func (s *ApplicationService) checkDuplicate(ctx context.Context, n model.NewApplication) error {
	duplicate, err := s.store.HasRecent(ctx, n)
	if err != nil {
		return err
	}
	if duplicate {
		return errDuplicateApplication
	}
	return nil
}

func (s *ApplicationService) insert(ctx context.Context, n model.NewApplication) (string, error) {
	for range publicid.MaxAttempts {
		pid, err := s.newID()
		if err != nil {
			return "", err
		}
		n.PublicID = pid
		err = s.store.Create(ctx, n)
		switch {
		case err == nil:
			return pid, nil
		case errors.Is(err, repository.ErrDuplicateApplication):
			return "", errDuplicateApplication
		case postgres.IsUniqueViolation(err, "applicants_public_id_key"):
			continue
		default:
			return "", err
		}
	}
	return "", fmt.Errorf("create application: no unique public id after %d attempts", publicid.MaxAttempts)
}

type pendingFile struct {
	field string
	kind  string
	file  ApplicationFile
}

func (s *ApplicationService) storeFiles(ctx context.Context, ownerID uuid.UUID, in ApplicationInput, limits ApplicationLimits) ([]model.NewApplicantFile, error) {
	pending := make([]pendingFile, 0, len(in.CV)+len(in.Documents))
	for _, f := range in.CV {
		pending = append(pending, pendingFile{field: "cv", kind: model.ApplicantFileCV, file: f})
	}
	for _, f := range in.Documents {
		pending = append(pending, pendingFile{field: "documents", kind: model.ApplicantFileDocument, file: f})
	}
	stored := make([]model.NewApplicantFile, len(pending))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(applicationUploadWorkers)
	for i, p := range pending {
		g.Go(func() error {
			id, err := s.upload(gctx, ownerID, p, limits.MaxFileBytes)
			if err != nil {
				return err
			}
			stored[i] = model.NewApplicantFile{FileID: id, Kind: p.kind}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		s.discard(ctx, stored)
		return nil, err
	}
	return stored, nil
}

func (s *ApplicationService) upload(ctx context.Context, ownerID uuid.UUID, p pendingFile, maxBytes int64) (uuid.UUID, error) {
	rc, err := p.file.Open()
	if err != nil {
		return uuid.Nil, fmt.Errorf("open uploaded file: %w", err)
	}
	data, err := io.ReadAll(io.LimitReader(rc, maxBytes+1))
	_ = rc.Close()
	if err != nil {
		return uuid.Nil, fmt.Errorf("read uploaded file: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return uuid.Nil, fieldError(filecheck.TooLarge(fmt.Sprintf("The file may not be larger than %d MB.", maxBytes>>20)), p.field)
	}
	f, err := s.files.Upload(ctx, UploadInput{Context: filecheck.ContextApplicant, Filename: p.file.Filename, Data: data, OwnerType: "applicant", OwnerID: &ownerID})
	if err != nil {
		return uuid.Nil, fieldError(err, p.field)
	}
	return f.ID, nil
}

func fieldError(err error, field string) error {
	e, ok := apperr.As(err)
	if !ok {
		return err
	}
	c := *e
	c.Fields = map[string]string{field: e.Message}
	return &c
}

func (s *ApplicationService) discard(ctx context.Context, files []model.NewApplicantFile) {
	detached := context.WithoutCancel(ctx)
	for _, f := range files {
		if f.FileID == uuid.Nil {
			continue
		}
		if err := s.files.Delete(detached, f.FileID); err != nil {
			logctx.From(ctx).WarnContext(ctx, "uploaded application file not removed", "file_id", f.FileID, "error", err)
		}
	}
}

func (s *ApplicationService) announce(ctx context.Context, n model.NewApplication, publicID string, v model.Vacancy) {
	ctx = context.WithoutCancel(ctx)
	log := logctx.From(ctx)
	if err := s.dashboard.Invalidate(ctx); err != nil {
		log.WarnContext(ctx, "dashboard cache not invalidated after application", "error", err)
	}
	vars := map[string]string{
		"application_id": publicID, "vacancy_title": v.Title, "vacancy_id": v.PublicID,
		"applicant_name": n.FullName, "applicant_email": n.Email, "admin_url": s.adminURL + "/applicants/" + publicID,
	}
	if _, err := s.mailer.SendTemplate(ctx, "applicant_confirmation", n.Email, vars, "applicant", &n.ID); err != nil {
		log.WarnContext(ctx, "applicant confirmation email not queued", "application_id", publicID, "error", err)
	}
	if err := s.mailer.Notify(ctx, "admin_new_application", vars, "applicant", &n.ID); err != nil {
		log.WarnContext(ctx, "admin application notification not queued", "application_id", publicID, "error", err)
	}
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "on", "yes":
		return true
	}
	return false
}

func cleanIP(raw string) string {
	addr, err := netip.ParseAddr(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return addr.String()
}

func plainText(s string, multiline bool) string {
	s = strings.Map(func(r rune) rune {
		if multiline && (r == '\n' || r == '\t') {
			return r
		}
		if r == '\r' {
			return -1
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}
