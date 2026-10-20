// Package admin holds controllers for the admin API.
package admin

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/middleware"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/token"
)

// AuthService is what the auth controller needs.
type AuthService interface {
	Login(ctx context.Context, email, password string, client service.Client, previousSessionID string) (service.LoginResult, error)
	Logout(ctx context.Context, sessionID string) error
	ChangePassword(ctx context.Context, adminID uuid.UUID, current, next string, client service.Client) (string, error)
	Recover(ctx context.Context, email, code, next string, client service.Client) (service.LoginResult, error)
	RegenerateRecoveryCodes(ctx context.Context, adminID uuid.UUID) ([]string, error)
	UpdateProfile(ctx context.Context, adminID uuid.UUID, name, email string) (model.Admin, error)
}

// AuthHandler serves login, logout, recovery and the current admin.
type AuthHandler struct {
	auth    AuthService
	cookies middleware.CookieWriter
}

// NewAuthHandler returns an AuthHandler.
func NewAuthHandler(auth AuthService, cookies middleware.CookieWriter) *AuthHandler {
	return &AuthHandler{auth: auth, cookies: cookies}
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in request.Login
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	res, err := h.auth.Login(r.Context(), in.Email, in.Password, clientOf(r), previousSession(r))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	h.respondWithSession(w, r, res.SessionID, res.Admin)
}

// Recover handles POST /auth/recovery.
func (h *AuthHandler) Recover(w http.ResponseWriter, r *http.Request) {
	var in request.Recovery
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	res, err := h.auth.Recover(r.Context(), in.Email, in.Code, in.NewPassword, clientOf(r))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	h.respondWithSession(w, r, res.SessionID, res.Admin)
}

// Logout handles POST /auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.auth.Logout(r.Context(), middleware.SessionID(r.Context())); err != nil {
		resource.Error(w, r, err)
		return
	}
	h.cookies.Clear(w)
	resource.NoContent(w)
}

// Me handles GET /me and returns the CSRF token for the SPA.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	admin, _ := middleware.Admin(r.Context())
	csrf, err := r.Cookie(middleware.CSRFCookie)
	if err != nil || csrf.Value == "" {
		h.respondWithSession(w, r, middleware.SessionID(r.Context()), admin)
		return
	}
	resource.JSON(w, http.StatusOK, resource.Session{Admin: resource.NewAdmin(admin), CSRFToken: csrf.Value})
}

// UpdateProfile handles PUT /me.
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var in request.UpdateProfile
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	admin, _ := middleware.Admin(r.Context())
	updated, err := h.auth.UpdateProfile(r.Context(), admin.ID, in.Name, in.Email)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewAdmin(updated))
}

// ChangePassword handles PUT /me/password and rotates the session.
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var in request.ChangePassword
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	admin, _ := middleware.Admin(r.Context())
	sid, err := h.auth.ChangePassword(r.Context(), admin.ID, in.CurrentPassword, in.NewPassword, clientOf(r))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	admin.MustChangePassword = false
	h.respondWithSession(w, r, sid, admin)
}

// RegenerateRecoveryCodes handles POST /recovery-codes/regenerate.
func (h *AuthHandler) RegenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	admin, _ := middleware.Admin(r.Context())
	codes, err := h.auth.RegenerateRecoveryCodes(r.Context(), admin.ID)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.RecoveryCodes{Codes: codes})
}

func (h *AuthHandler) respondWithSession(w http.ResponseWriter, r *http.Request, sessionID string, admin model.Admin) {
	csrf, err := token.URLSafe(32)
	if err != nil {
		resource.Error(w, r, apperr.ErrInternal.Wrap(err))
		return
	}
	h.cookies.SetSession(w, sessionID, csrf)
	resource.JSON(w, http.StatusOK, resource.Session{Admin: resource.NewAdmin(admin), CSRFToken: csrf})
}

func clientOf(r *http.Request) service.Client {
	ua := r.UserAgent()
	if len(ua) > 512 {
		ua = ua[:512]
	}
	return service.Client{IP: middleware.ClientIP(r.Context()), UserAgent: ua}
}

func previousSession(r *http.Request) string {
	c, err := r.Cookie(middleware.SessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}
