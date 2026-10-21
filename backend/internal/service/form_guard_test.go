package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/platform/turnstile"
)

type scriptedVerifier struct {
	res    turnstile.Result
	err    error
	called int
	secret string
	token  string
	ip     string
}

func (v *scriptedVerifier) Verify(_ context.Context, secret, token, ip string) (turnstile.Result, error) {
	v.called++
	v.secret, v.token, v.ip = secret, token, ip
	return v.res, v.err
}

func TestFormGuardVerify(t *testing.T) {
	configured := staticSettings{"security": {"turnstile_secret_key": "secret"}}
	unconfigured := staticSettings{"security": {"turnstile_secret_key": ""}}
	tests := []struct {
		name          string
		settings      staticSettings
		allowLocal    bool
		token         string
		verifier      scriptedVerifier
		wantCode      string
		wantStatus    int
		wantVerifyHit bool
	}{
		{"valid token", configured, false, "tok", scriptedVerifier{res: turnstile.Result{Success: true}}, "", 0, true},
		{"rejected token", configured, false, "tok", scriptedVerifier{res: turnstile.Result{ErrorCodes: []string{"invalid-input-response"}}}, "turnstile_failed", 422, true},
		{"missing token", configured, false, "  ", scriptedVerifier{res: turnstile.Result{Success: true}}, "turnstile_failed", 422, false},
		{"cloudflare unreachable", configured, false, "tok", scriptedVerifier{err: errors.New("timeout")}, "turnstile_unavailable", 503, true},
		{"no secret outside local", unconfigured, false, "tok", scriptedVerifier{res: turnstile.Result{Success: true}}, "turnstile_unconfigured", 503, false},
		{"no secret in local", unconfigured, true, "", scriptedVerifier{}, "", 0, false},
		{"secret set in local still verifies", configured, true, "", scriptedVerifier{res: turnstile.Result{Success: true}}, "turnstile_failed", 422, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := tc.verifier
			err := NewFormGuard(&v, tc.settings, tc.allowLocal).Verify(context.Background(), tc.token, "203.0.113.9")

			if tc.wantCode == "" {
				require.NoError(t, err)
			} else {
				requireAppError(t, err, tc.wantStatus, tc.wantCode)
			}
			assert.Equal(t, tc.wantVerifyHit, v.called == 1)
			if tc.wantVerifyHit {
				assert.Equal(t, "secret", v.secret)
				assert.Equal(t, "203.0.113.9", v.ip)
			}
		})
	}
}
