package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// Cookie and header names.
const (
	SessionCookie = "sc_session"
	CSRFCookie    = "sc_csrf"
	CSRFHeader    = "X-CSRF-Token"
)

// Authenticator resolves a session ID to an admin.
type Authenticator interface {
	Authenticate(ctx context.Context, sessionID string) (model.Admin, error)
}

// RequireSession rejects requests without a valid session.
func RequireSession(auth Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(SessionCookie)
			if err != nil || c.Value == "" {
				resource.Error(w, r, apperr.ErrUnauthenticated)
				return
			}
			admin, err := auth.Authenticate(r.Context(), c.Value)
			if err != nil {
				resource.Error(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithAdmin(r.Context(), admin, c.Value)))
		})
	}
}

// RequirePasswordCurrent blocks every route except the allowed ones while a password change is pending.
func RequirePasswordCurrent(allowed ...string) func(http.Handler) http.Handler {
	set := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		set[a] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			admin, ok := Admin(r.Context())
			if ok && admin.MustChangePassword {
				if _, allow := set[r.Method+" "+r.URL.Path]; !allow {
					resource.Error(w, r, apperr.ErrPasswordChangeRequired)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CSRF enforces the double-submit token on state-changing requests.
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		c, err := r.Cookie(CSRFCookie)
		header := r.Header.Get(CSRFHeader)
		if err != nil || c.Value == "" || header == "" || subtle.ConstantTimeCompare([]byte(c.Value), []byte(header)) != 1 {
			resource.Error(w, r, apperr.ErrCSRF)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CookieWriter sets and clears auth cookies with consistent attributes.
type CookieWriter struct {
	Secure  bool
	MaxAge  time.Duration
	Session string
	CSRF    string
}

// NewCookieWriter returns a CookieWriter.
func NewCookieWriter(secure bool, maxAge time.Duration) CookieWriter {
	return CookieWriter{Secure: secure, MaxAge: maxAge, Session: SessionCookie, CSRF: CSRFCookie}
}

// SetSession writes the session and CSRF cookies.
func (c CookieWriter) SetSession(w http.ResponseWriter, sessionID, csrfToken string) {
	http.SetCookie(w, c.cookie(c.Session, sessionID, int(c.MaxAge.Seconds())))
	http.SetCookie(w, c.cookie(c.CSRF, csrfToken, int(c.MaxAge.Seconds())))
}

// Clear expires both cookies.
func (c CookieWriter) Clear(w http.ResponseWriter) {
	http.SetCookie(w, c.cookie(c.Session, "", -1))
	http.SetCookie(w, c.cookie(c.CSRF, "", -1))
}

func (c CookieWriter) cookie(name, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: http.SameSiteStrictMode,
	}
}
