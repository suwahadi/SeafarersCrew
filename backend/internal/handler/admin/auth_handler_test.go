package admin

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

	"github.com/suwahadi/seafarerscrew/backend/internal/middleware"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type stubAuth struct {
	admin model.Admin
}

func (s stubAuth) Login(_ context.Context, email, pw string, _ service.Client, _ string) (service.LoginResult, error) {
	if email == s.admin.Email && pw == "right-password" {
		return service.LoginResult{SessionID: "new-session", Admin: s.admin}, nil
	}
	return service.LoginResult{}, apperr.ErrInvalidCredentials
}

func (stubAuth) Logout(context.Context, string) error { return nil }

func (stubAuth) ChangePassword(context.Context, uuid.UUID, string, string, service.Client) (string, error) {
	return "rotated", nil
}

func (stubAuth) Recover(context.Context, string, string, string, service.Client) (service.LoginResult, error) {
	return service.LoginResult{}, apperr.ErrInvalidCredentials
}

func (stubAuth) RegenerateRecoveryCodes(context.Context, uuid.UUID) ([]string, error) {
	return []string{"AAAAA-BBBBB"}, nil
}

func (s stubAuth) UpdateProfile(context.Context, uuid.UUID, string, string) (model.Admin, error) {
	return s.admin, nil
}

func newHandler() *AuthHandler {
	admin := model.Admin{ID: uuid.New(), Name: "Ambon", Email: "admin@example.com", PasswordHash: "$argon2id$secret"}
	return NewAuthHandler(stubAuth{admin: admin}, middleware.NewCookieWriter(true, time.Hour))
}

func TestLoginSetsCookiesAndReturnsEnvelope(t *testing.T) {
	h := newHandler()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"admin@example.com","password":"right-password"}`))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data struct {
			Admin struct {
				Email string `json:"email"`
			} `json:"admin"`
			CSRFToken string `json:"csrf_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "admin@example.com", body.Data.Admin.Email)
	assert.NotEmpty(t, body.Data.CSRFToken)
	assert.NotContains(t, rec.Body.String(), "argon2id", "password hash must never be serialized")

	cookies := map[string]*http.Cookie{}
	for _, c := range rec.Result().Cookies() {
		cookies[c.Name] = c
	}
	assert.Equal(t, "new-session", cookies[middleware.SessionCookie].Value)
	assert.Equal(t, body.Data.CSRFToken, cookies[middleware.CSRFCookie].Value)
}

func TestLoginErrors(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		status int
		code   string
	}{
		{"wrong password", `{"email":"admin@example.com","password":"nope"}`, http.StatusUnauthorized, "invalid_credentials"},
		{"missing fields", `{}`, http.StatusUnprocessableEntity, "validation_failed"},
		{"malformed json", `{"email":`, http.StatusBadRequest, "bad_request"},
		{"empty body", ``, http.StatusBadRequest, "bad_request"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newHandler().Login(rec, httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(tc.body)))
			assert.Equal(t, tc.status, rec.Code)
			assert.Contains(t, rec.Body.String(), `"code":"`+tc.code+`"`)
			assert.Empty(t, rec.Result().Cookies())
		})
	}
}

func TestLoginRejectsOversizedBody(t *testing.T) {
	rec := httptest.NewRecorder()
	body := `{"email":"` + strings.Repeat("a", 2<<20) + `"}`
	newHandler().Login(rec, httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body)))
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestLogoutClearsCookies(t *testing.T) {
	rec := httptest.NewRecorder()
	newHandler().Logout(rec, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))
	assert.Equal(t, http.StatusNoContent, rec.Code)
	for _, c := range rec.Result().Cookies() {
		assert.Less(t, c.MaxAge, 0)
	}
}
