package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

var ok = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body.Error.Code
}

type fakeAuth struct {
	admin model.Admin
}

func (f fakeAuth) Authenticate(_ context.Context, sid string) (model.Admin, error) {
	if sid != "good" {
		return model.Admin{}, apperr.ErrUnauthenticated
	}
	return f.admin, nil
}

func TestRequireSession(t *testing.T) {
	h := RequireSession(fakeAuth{admin: model.Admin{ID: uuid.New()}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, found := Admin(r.Context())
		assert.True(t, found)
		assert.Equal(t, "good", SessionID(r.Context()))
		w.WriteHeader(http.StatusOK)
	}))
	cases := []struct {
		name   string
		cookie *http.Cookie
		status int
	}{
		{"no cookie", nil, http.StatusUnauthorized},
		{"empty cookie", &http.Cookie{Name: SessionCookie, Value: ""}, http.StatusUnauthorized},
		{"unknown session", &http.Cookie{Name: SessionCookie, Value: "bad"}, http.StatusUnauthorized},
		{"valid session", &http.Cookie{Name: SessionCookie, Value: "good"}, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/crews", nil)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			assert.Equal(t, tc.status, rec.Code)
			if tc.status == http.StatusUnauthorized {
				assert.Equal(t, "unauthenticated", errorCode(t, rec))
				assert.NotContains(t, rec.Body.String(), "admin@")
			}
		})
	}
}

func TestCSRFRequiresMatchingHeaderOnUnsafeMethods(t *testing.T) {
	h := CSRF(ok)
	cases := []struct {
		name   string
		method string
		cookie string
		header string
		status int
	}{
		{"GET is exempt", http.MethodGet, "", "", http.StatusOK},
		{"POST without token", http.MethodPost, "", "", http.StatusForbidden},
		{"POST with cookie only", http.MethodPost, "abc", "", http.StatusForbidden},
		{"POST with header only", http.MethodPost, "", "abc", http.StatusForbidden},
		{"POST with mismatch", http.MethodPost, "abc", "abd", http.StatusForbidden},
		{"PUT with match", http.MethodPut, "abc", "abc", http.StatusOK},
		{"DELETE with match", http.MethodDelete, "abc", "abc", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/", nil)
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: CSRFCookie, Value: tc.cookie})
			}
			if tc.header != "" {
				req.Header.Set(CSRFHeader, tc.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			assert.Equal(t, tc.status, rec.Code)
		})
	}
}

func TestRequirePasswordCurrentBlocksOtherRoutesWhileChangeIsPending(t *testing.T) {
	h := RequirePasswordCurrent("GET /me", "PUT /me/password")(ok)
	run := func(mustChange bool, method, path string) int {
		req := httptest.NewRequest(method, path, nil)
		req = req.WithContext(WithAdmin(req.Context(), model.Admin{MustChangePassword: mustChange}, "s"))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	assert.Equal(t, http.StatusOK, run(true, http.MethodGet, "/me"))
	assert.Equal(t, http.StatusOK, run(true, http.MethodPut, "/me/password"))
	assert.Equal(t, http.StatusForbidden, run(true, http.MethodGet, "/crews"))
	assert.Equal(t, http.StatusOK, run(false, http.MethodGet, "/crews"))
}

type fakeCounter struct {
	n map[string]int64
}

func (f *fakeCounter) Hit(_ context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	f.n[key]++
	return f.n[key], window, nil
}

func TestRateLimitPerIP(t *testing.T) {
	counter := &fakeCounter{n: map[string]int64{}}
	h := RealIP(RateLimit(counter, "form", 5, 10*time.Minute)(ok))
	send := func(ip string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/contact", nil)
		req.RemoteAddr = ip + ":5555"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	for i := 0; i < 5; i++ {
		assert.Equal(t, http.StatusOK, send("203.0.113.1").Code)
	}
	blocked := send("203.0.113.1")
	assert.Equal(t, http.StatusTooManyRequests, blocked.Code)
	assert.Equal(t, "600", blocked.Header().Get("Retry-After"))
	assert.Equal(t, http.StatusOK, send("203.0.113.2").Code)
}

func TestRealIPUsesRightmostForwardedAddress(t *testing.T) {
	var got string
	h := RealIP(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = ClientIP(r.Context()) }))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.18.0.5:443"
	req.Header.Set("X-Forwarded-For", "1.1.1.1, 198.51.100.9")
	h.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "198.51.100.9", got)

	req.Header.Set("X-Forwarded-For", "garbage")
	h.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "172.18.0.5", got)
}

func TestCORSAllowsOnlyConfiguredOrigin(t *testing.T) {
	h := CORS("https://admin.example.com")(ok)

	pre := httptest.NewRequest(http.MethodOptions, "/", nil)
	pre.Header.Set("Origin", "https://evil.example.com")
	pre.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, pre)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	pre.Header.Set("Origin", "https://admin.example.com")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, pre)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "https://admin.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))

	allowed := httptest.NewRequest(http.MethodGet, "/", nil)
	allowed.Header.Set("Origin", "https://admin.example.com")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, allowed)
	assert.Equal(t, "X-Request-ID, Retry-After", rec.Header().Get("Access-Control-Expose-Headers"), "browsers need Retry-After to show when a form may be sent again")

	get := httptest.NewRequest(http.MethodGet, "/", nil)
	get.Header.Set("Origin", "https://evil.example.com")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, get)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestRequireJSON(t *testing.T) {
	h := RequireJSON(ok)
	for ct, want := range map[string]int{
		"application/json":                  http.StatusOK,
		"application/json; charset=utf-8":   http.StatusOK,
		"application/x-www-form-urlencoded": http.StatusUnsupportedMediaType,
		"text/plain":                        http.StatusUnsupportedMediaType,
		"":                                  http.StatusUnsupportedMediaType,
	} {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
		req.Header.Set("Content-Type", ct)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, want, rec.Code, ct)
	}
}

func TestRecoverReturnsGenericError(t *testing.T) {
	h := Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("passport X1234567 leaked") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "internal_error", errorCode(t, rec))
	assert.NotContains(t, rec.Body.String(), "X1234567")
}

func TestCookieWriterSetsHardenedCookies(t *testing.T) {
	rec := httptest.NewRecorder()
	NewCookieWriter(true, time.Hour).SetSession(rec, "sid", "csrf")
	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 2)
	for _, c := range cookies {
		assert.True(t, c.HttpOnly)
		assert.True(t, c.Secure)
		assert.Equal(t, http.SameSiteStrictMode, c.SameSite)
		assert.Equal(t, "/", c.Path)
	}
}

func TestRequestIDIsGeneratedOrPropagated(t *testing.T) {
	var seen string
	h2 := RequestID(testLogger())(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = RequestIDFrom(r.Context()) }))

	rec := httptest.NewRecorder()
	h2.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.NotEmpty(t, seen)
	assert.Equal(t, seen, rec.Header().Get("X-Request-ID"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "abc12345-trace")
	rec = httptest.NewRecorder()
	h2.ServeHTTP(rec, req)
	assert.Equal(t, "abc12345-trace", seen)

	req.Header.Set("X-Request-ID", "<script>")
	h2.ServeHTTP(httptest.NewRecorder(), req)
	assert.NotEqual(t, "<script>", seen)
}

func TestAfterWriteRunsOnlyAfterSuccessfulWrites(t *testing.T) {
	calls := 0
	mw := AfterWrite(func(context.Context) { calls++ })
	run := func(method string, status int) {
		h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, "/", nil))
	}
	run(http.MethodGet, http.StatusOK)
	run(http.MethodPost, http.StatusUnprocessableEntity)
	assert.Equal(t, 0, calls)
	run(http.MethodPost, http.StatusCreated)
	run(http.MethodDelete, http.StatusNoContent)
	assert.Equal(t, 2, calls)
}

func TestTimeoutWithUploadsGivesUploadPostsTheLongerDeadline(t *testing.T) {
	var got time.Duration
	h := TimeoutWithUploads(time.Second, time.Hour, "/apply")(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		dl, _ := r.Context().Deadline()
		got = time.Until(dl)
	}))
	cases := []struct {
		method, path string
		long         bool
	}{
		{http.MethodPost, "/apply", true},
		{http.MethodGet, "/apply", false},
		{http.MethodPost, "/contact", false},
	}
	for _, c := range cases {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(c.method, c.path, nil))
		assert.Equal(t, c.long, got > time.Minute, c.method+" "+c.path)
	}
}
