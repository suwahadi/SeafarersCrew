package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/public"
	"github.com/suwahadi/seafarerscrew/backend/internal/middleware"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type hit struct {
	key    string
	window time.Duration
}

type recordingCounter struct {
	mu     sync.Mutex
	counts map[string]int64
	hits   []hit
}

func (c *recordingCounter) Hit(_ context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.counts == nil {
		c.counts = map[string]int64{}
	}
	c.counts[key]++
	c.hits = append(c.hits, hit{key: key, window: window})
	return c.counts[key], window, nil
}

type memCache struct {
	mu     sync.Mutex
	values map[string][]byte
}

func (m *memCache) TagVersions(context.Context, []string) (string, error) { return "v1", nil }

func (m *memCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.values[key]
	return v, ok, nil
}

func (m *memCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.values == nil {
		m.values = map[string][]byte{}
	}
	m.values[key] = value
	return nil
}

type okApplications struct{}

func (okApplications) Limits(context.Context) (service.ApplicationLimits, error) {
	return service.ApplicationLimits{MaxDocuments: 3, MaxFileBytes: 1 << 20}, nil
}

func (okApplications) Submit(context.Context, service.ApplicationInput) (string, error) {
	return "APP222222", nil
}

type okContact struct{}

func (okContact) Submit(context.Context, service.ContactInput) error { return nil }

type countingVacancies struct{ lists int }

func (c *countingVacancies) PublicList(context.Context, model.PublicVacancyQuery) (repository.Page[model.Vacancy], error) {
	c.lists++
	return repository.Page[model.Vacancy]{}, nil
}

func (c *countingVacancies) PublicDetail(context.Context, string) (service.PublicVacancy, error) {
	return service.PublicVacancy{}, apperr.NotFound("Vacancy")
}

func (c *countingVacancies) PublicFilters(context.Context) (model.VacancyFilterOptions, error) {
	return model.VacancyFilterOptions{}, nil
}

func publicRouter(counter *recordingCounter, vacancies *countingVacancies) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	Public(r, PublicDeps{
		Counter: counter,
		Cache:   &memCache{},
		Career: PublicCareerRoutes{
			Vacancies:    public.NewVacancyHandler(vacancies),
			Applications: public.NewApplicationHandler(okApplications{}),
			Contact:      public.NewContactHandler(okContact{}),
		},
	})
	return r
}

func postForm(h http.Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, PublicPrefix+path, strings.NewReader("--b\r\nContent-Disposition: form-data; name=\"website\"\r\n\r\n\r\n--b--\r\n"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=b")
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func postJSON(h http.Handler, path, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, PublicPrefix+path, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", ip)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestApplicationsAreLimitedToFivePerTenMinutesPerIP(t *testing.T) {
	counter := &recordingCounter{}
	h := publicRouter(counter, &countingVacancies{})

	for i := range 5 {
		require.Equal(t, http.StatusCreated, postForm(h, "/applications").Code, "submission %d", i+1)
	}
	rec := postForm(h, "/applications")

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
	assert.Equal(t, int64(6), counter.counts["applications:203.0.113.9"])
	assert.Equal(t, int64(6), counter.counts["public-api:203.0.113.9"], "the general public limiter applies as well")
	for _, hh := range counter.hits {
		if strings.HasPrefix(hh.key, "applications:") {
			assert.Equal(t, 10*time.Minute, hh.window)
		}
	}
}

func TestContactAndApplicationBucketsAreIndependentPerForm(t *testing.T) {
	counter := &recordingCounter{}
	h := publicRouter(counter, &countingVacancies{})
	for range 6 {
		postForm(h, "/applications")
	}

	assert.Equal(t, http.StatusCreated, postJSON(h, "/contact", "203.0.113.9").Code, "an exhausted application bucket does not block the contact form")
	for range 4 {
		require.Equal(t, http.StatusCreated, postJSON(h, "/contact", "203.0.113.9").Code)
	}
	assert.Equal(t, http.StatusTooManyRequests, postJSON(h, "/contact", "203.0.113.9").Code)
	assert.Equal(t, http.StatusCreated, postJSON(h, "/contact", "203.0.113.10").Code, "limits are per IP")
}

func TestContactRejectsNonJSONBodies(t *testing.T) {
	h := publicRouter(&recordingCounter{}, &countingVacancies{})
	req := httptest.NewRequest(http.MethodPost, PublicPrefix+"/contact", strings.NewReader("name=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
}

func TestVacancyReadsAreCachedButFormsAreNot(t *testing.T) {
	vacancies := &countingVacancies{}
	h := publicRouter(&recordingCounter{}, vacancies)
	get := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, PublicPrefix+"/vacancies?q=master", nil))
		return rec
	}

	first, second := get(), get()

	assert.Equal(t, "MISS", first.Header().Get("X-Cache"))
	assert.Equal(t, "HIT", second.Header().Get("X-Cache"))
	assert.Equal(t, 1, vacancies.lists)
	assert.Equal(t, http.StatusNotFound, func() int {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, PublicPrefix+"/vacancies/draft-one", nil))
		return rec.Code
	}())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, PublicPrefix+"/vacancy-filters", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
}

type rejectingAuth struct{}

func (rejectingAuth) Authenticate(context.Context, string) (model.Admin, error) {
	return model.Admin{}, apperr.ErrUnauthenticated
}

func TestAdminCareerEndpointsRequireASession(t *testing.T) {
	r := chi.NewRouter()
	Admin(r, AdminDeps{Origin: "https://admin.example.com", Authenticator: rejectingAuth{}, Counter: &recordingCounter{}})
	endpoints := []struct{ method, path string }{
		{http.MethodGet, "/vacancies"}, {http.MethodPost, "/vacancies"}, {http.MethodGet, "/vacancies/VAC222222"},
		{http.MethodPut, "/vacancies/VAC222222"}, {http.MethodDelete, "/vacancies/VAC222222"}, {http.MethodPatch, "/vacancies/VAC222222/status"},
		{http.MethodGet, "/applicants"}, {http.MethodGet, "/applicants/stage-counts"}, {http.MethodGet, "/applicants/APP222222"},
		{http.MethodDelete, "/applicants/APP222222"}, {http.MethodPatch, "/applicants/APP222222/stage"},
		{http.MethodGet, "/contact-messages"}, {http.MethodGet, "/contact-messages/unread-count"},
		{http.MethodGet, "/contact-messages/6f1f4c0e-1111-7222-8333-944455556666"}, {http.MethodPatch, "/contact-messages/6f1f4c0e-1111-7222-8333-944455556666"},
	}
	for _, e := range endpoints {
		t.Run(e.method+" "+e.path, func(t *testing.T) {
			req := httptest.NewRequest(e.method, AdminPrefix+e.path, strings.NewReader(`{}`))
			req.AddCookie(&http.Cookie{Name: middleware.SessionCookie, Value: "stale"})
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}
