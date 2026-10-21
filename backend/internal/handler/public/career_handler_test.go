package public

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

type stubApplications struct {
	in     service.ApplicationInput
	files  map[string]string
	err    error
	limits service.ApplicationLimits
}

func (s *stubApplications) Limits(context.Context) (service.ApplicationLimits, error) {
	return s.limits, nil
}

func (s *stubApplications) Submit(_ context.Context, in service.ApplicationInput) (string, error) {
	s.in = in
	s.files = map[string]string{}
	for field, list := range map[string][]service.ApplicationFile{"cv": in.CV, "documents": in.Documents} {
		for _, f := range list {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			data, _ := io.ReadAll(rc)
			_ = rc.Close()
			s.files[field+":"+f.Filename] = string(data)
		}
	}
	return "APP222222", s.err
}

func applicationRouter(s *stubApplications) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Post("/applications", NewApplicationHandler(s).Store)
	return r
}

func multipartBody(t *testing.T, fields map[string]string, files map[string][]string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}
	for field, names := range files {
		for _, name := range names {
			part, err := w.CreateFormFile(field, name)
			require.NoError(t, err)
			_, err = part.Write([]byte("content of " + name))
			require.NoError(t, err)
		}
	}
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func postApplication(h http.Handler, body io.Reader, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/applications", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestApplicationStorePassesFieldsAndFilesToService(t *testing.T) {
	s := &stubApplications{limits: service.ApplicationLimits{MaxDocuments: 5, MaxFileBytes: 1 << 20}}
	body, ct := multipartBody(t, map[string]string{
		"vacancy": "chief-officer", "full_name": "Siti Aminah", "email": "siti@example.com", "phone": "+6281234567890",
		"date_of_birth": "1990-05-01", "nationality": "Indonesian", "rank_applied_id": "r1", "years_experience": "8",
		"message": "Hello", "consent": "true", "website": "", "cf-turnstile-response": "tok",
	}, map[string][]string{"cv": {"cv.pdf"}, "documents": {"a.pdf", "b.png"}})

	rec := postApplication(applicationRouter(s), body, ct)

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"data":{"application_id":"APP222222"}}`, rec.Body.String())
	assert.Equal(t, "chief-officer", s.in.Vacancy)
	assert.Equal(t, "Siti Aminah", s.in.FullName)
	assert.Equal(t, "tok", s.in.TurnstileToken)
	assert.Equal(t, "203.0.113.9", s.in.IP)
	assert.Equal(t, "true", s.in.Consent)
	assert.Equal(t, map[string]string{"cv:cv.pdf": "content of cv.pdf", "documents:a.pdf": "content of a.pdf", "documents:b.png": "content of b.png"}, s.files)
}

func TestApplicationStoreForwardsHoneypotValue(t *testing.T) {
	s := &stubApplications{limits: service.ApplicationLimits{MaxDocuments: 5, MaxFileBytes: 1 << 20}}
	body, ct := multipartBody(t, map[string]string{"website": "http://spam.example"}, nil)

	rec := postApplication(applicationRouter(s), body, ct)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "http://spam.example", s.in.Website)
}

func TestApplicationStoreRejectsNonMultipartBodies(t *testing.T) {
	s := &stubApplications{}
	rec := postApplication(applicationRouter(s), strings.NewReader(`{"vacancy":"x"}`), "application/json")
	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
}

func TestApplicationStoreRejectsBodiesAboveTheFileLimits(t *testing.T) {
	s := &stubApplications{limits: service.ApplicationLimits{MaxDocuments: 1, MaxFileBytes: 1024}}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("cv", "cv.pdf")
	require.NoError(t, err)
	_, err = part.Write(bytes.Repeat([]byte("a"), 2<<20))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	rec := postApplication(applicationRouter(s), &buf, w.FormDataContentType())

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Contains(t, rec.Body.String(), "payload_too_large")
}

func TestApplicationStoreShowsValidationFieldsFromService(t *testing.T) {
	s := &stubApplications{
		limits: service.ApplicationLimits{MaxDocuments: 5, MaxFileBytes: 1 << 20},
		err:    apperr.Validation(map[string]string{"email": "This must be a valid email address."}),
	}
	body, ct := multipartBody(t, map[string]string{"email": "x"}, nil)

	rec := postApplication(applicationRouter(s), body, ct)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var out struct {
		Error struct {
			Code   string            `json:"code"`
			Fields map[string]string `json:"fields"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	assert.Equal(t, "validation_failed", out.Error.Code)
	assert.Contains(t, out.Error.Fields, "email")
}

func TestApplicationStoreMapsDuplicateToConflict(t *testing.T) {
	s := &stubApplications{limits: service.ApplicationLimits{MaxDocuments: 5, MaxFileBytes: 1 << 20}, err: apperr.Conflict("duplicate_application", "You have already applied.")}
	body, ct := multipartBody(t, nil, nil)

	rec := postApplication(applicationRouter(s), body, ct)

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "duplicate_application")
}

type stubContact struct {
	in  service.ContactInput
	err error
}

func (s *stubContact) Submit(_ context.Context, in service.ContactInput) error {
	s.in = in
	return s.err
}

func contactRouter(s *stubContact) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Post("/contact", NewContactHandler(s).Store)
	return r
}

func TestContactStorePassesFieldsAndTurnstileTokenToService(t *testing.T) {
	s := &stubContact{}
	body := `{"name":"John","email":"john@example.com","phone":"+6281234567890","subject":"Hi","message":"Hello","website":"","cf-turnstile-response":"tok"}`
	req := httptest.NewRequest(http.MethodPost, "/contact", strings.NewReader(body))
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	rec := httptest.NewRecorder()

	contactRouter(s).ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.JSONEq(t, `{"data":{"received":true}}`, rec.Body.String())
	assert.Equal(t, service.ContactInput{Name: "John", Email: "john@example.com", Phone: "+6281234567890", Subject: "Hi", Message: "Hello",
		TurnstileToken: "tok", IP: "203.0.113.9"}, s.in)
}

func TestContactStoreRejectsMalformedJSONAndPropagatesServiceErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	contactRouter(&stubContact{}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/contact", strings.NewReader(`{`)))
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	s := &stubContact{err: service.ErrTurnstileFailed}
	contactRouter(s).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/contact", strings.NewReader(`{}`)))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "turnstile_failed")
}

type stubVacancies struct {
	query  model.PublicVacancyQuery
	detail service.PublicVacancy
	err    error
}

func (s *stubVacancies) PublicList(_ context.Context, q model.PublicVacancyQuery) (repository.Page[model.Vacancy], error) {
	s.query = q
	closing := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)
	v := model.Vacancy{PublicID: "VAC222222", Title: "Chief Officer", Slug: "chief-officer", RankID: uuid.New(), RankName: "Chief Officer",
		Department: "deck", Location: "Singapore", EmploymentType: "contract", ClosingDate: &closing, DescriptionHTML: "<p>secret list payload</p>"}
	return repository.Page[model.Vacancy]{Items: []model.Vacancy{v}, Total: 31}, s.err
}

func (s *stubVacancies) PublicDetail(context.Context, string) (service.PublicVacancy, error) {
	return s.detail, s.err
}

func (s *stubVacancies) PublicFilters(context.Context) (model.VacancyFilterOptions, error) {
	return model.VacancyFilterOptions{Ranks: []model.Option{{ID: uuid.New(), Name: "Master", ShortCode: "MST"}}}, s.err
}

func vacancyRouter(s *stubVacancies) http.Handler {
	h := NewVacancyHandler(s)
	r := chi.NewRouter()
	r.Get("/vacancies", h.Index)
	r.Get("/vacancies/{slug}", h.Show)
	r.Get("/vacancy-filters", h.Filters)
	return r
}

func TestVacancyIndexParsesFiltersAndReturnsCards(t *testing.T) {
	s := &stubVacancies{}
	rank, vessel := uuid.New(), uuid.New()
	url := "/vacancies?q=%20engineer%20&rank=" + rank.String() + "&vessel_type=" + vessel.String() + "&department=deck&location=Singapore&page=2&per_page=500"
	rec := httptest.NewRecorder()

	vacancyRouter(s).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "engineer", s.query.Q)
	assert.Equal(t, rank, *s.query.RankID)
	assert.Equal(t, vessel, *s.query.VesselTypeID)
	assert.Equal(t, "deck", s.query.Department)
	assert.Equal(t, "Singapore", s.query.Location)
	assert.Equal(t, 2, s.query.Page)
	assert.Equal(t, 48, s.query.PerPage, "page size is capped")
	var out struct {
		Data []map[string]any `json:"data"`
		Meta resourceMeta     `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Len(t, out.Data, 1)
	assert.Equal(t, resourceMeta{Page: 2, PerPage: 48, Total: 31}, out.Meta)
	card := out.Data[0]
	for _, key := range []string{"id", "title", "slug", "rank", "vessel_type", "department", "location", "employment_type", "closing_date"} {
		assert.Contains(t, card, key)
	}
	assert.Equal(t, "2026-11-30", card["closing_date"])
	assert.NotContains(t, card, "description_html", "cards do not carry the description")
}

type resourceMeta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

func TestVacancyIndexDefaultsAndValidation(t *testing.T) {
	s := &stubVacancies{}
	rec := httptest.NewRecorder()
	vacancyRouter(s).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/vacancies", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, s.query.Page)
	assert.Equal(t, 12, s.query.PerPage)

	for _, url := range []string{"/vacancies?rank=not-a-uuid", "/vacancies?vessel_type=x", "/vacancies?department=bridge"} {
		rec = httptest.NewRecorder()
		vacancyRouter(s).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, url)
	}
}

func TestVacancyShowReturnsDetailWithOpenFlag(t *testing.T) {
	s := &stubVacancies{detail: service.PublicVacancy{
		Vacancy: model.Vacancy{PublicID: "VAC222222", Slug: "chief-officer", DescriptionHTML: "<p>x</p>", RequiredDocuments: []string{"Passport"}},
		IsOpen:  false,
	}}
	rec := httptest.NewRecorder()

	vacancyRouter(s).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/vacancies/chief-officer", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	var out struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	assert.Equal(t, false, out.Data["is_open"])
	assert.Equal(t, "<p>x</p>", out.Data["description_html"])
	assert.Equal(t, []any{"Passport"}, out.Data["required_documents"])

	s.err = apperr.NotFound("Vacancy")
	rec = httptest.NewRecorder()
	vacancyRouter(s).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/vacancies/draft", nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestVacancyFiltersReturnsEmptyListsNotNull(t *testing.T) {
	rec := httptest.NewRecorder()

	vacancyRouter(&stubVacancies{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/vacancy-filters", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	var out struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	assert.Len(t, out.Data["ranks"], 1)
	assert.Equal(t, []any{}, out.Data["departments"])
	assert.Equal(t, []any{}, out.Data["vessel_types"])
	assert.Equal(t, []any{}, out.Data["locations"])
}
