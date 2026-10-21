package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/suwahadi/seafarerscrew/backend/internal/platform/turnstile"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/logctx"
)

// Turnstile errors returned to public forms.
var (
	ErrTurnstileFailed       = apperr.New(http.StatusUnprocessableEntity, "turnstile_failed", "The verification check failed. Please try again.")
	ErrTurnstileUnconfigured = apperr.New(http.StatusServiceUnavailable, "turnstile_unconfigured", "Form verification is not configured.")
	ErrTurnstileUnavailable  = apperr.New(http.StatusServiceUnavailable, "turnstile_unavailable", "The verification service is unavailable. Please try again later.")
)

// TurnstileVerifier checks Turnstile tokens with Cloudflare.
type TurnstileVerifier interface {
	Verify(ctx context.Context, secret, token, ip string) (turnstile.Result, error)
}

// FormGuard verifies the Turnstile token of public form submissions (PRD 8.4).
type FormGuard struct {
	verifier          TurnstileVerifier
	settings          SettingsReader
	allowUnconfigured bool
}

// NewFormGuard builds a FormGuard; allowUnconfigured lets submissions through when no secret key is set, for local development.
func NewFormGuard(verifier TurnstileVerifier, settings SettingsReader, allowUnconfigured bool) *FormGuard {
	return &FormGuard{verifier: verifier, settings: settings, allowUnconfigured: allowUnconfigured}
}

// Verify accepts the submission only when Cloudflare confirms token for ip.
func (g *FormGuard) Verify(ctx context.Context, token, ip string) error {
	values, err := g.settings.Values(ctx, "security")
	if err != nil {
		return fmt.Errorf("read security settings: %w", err)
	}
	secret, _ := values["turnstile_secret_key"].(string)
	if secret == "" {
		if !g.allowUnconfigured {
			return ErrTurnstileUnconfigured
		}
		logctx.From(ctx).WarnContext(ctx, "turnstile secret key is not configured; accepting the form without verification")
		return nil
	}
	if strings.TrimSpace(token) == "" {
		return ErrTurnstileFailed
	}
	res, err := g.verifier.Verify(ctx, secret, token, ip)
	if err != nil {
		logctx.From(ctx).WarnContext(ctx, "turnstile verification failed to run", "error", err)
		return ErrTurnstileUnavailable
	}
	if !res.Success {
		return ErrTurnstileFailed
	}
	return nil
}
