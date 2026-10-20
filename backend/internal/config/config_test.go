package config

import (
	"encoding/base64"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validEnv() map[string]string {
	return map[string]string{
		"APP_ENV":            "local",
		"APP_URL":            "http://localhost:3000",
		"ADMIN_URL":          "http://localhost:5173/",
		"API_URL":            "http://localhost:8080",
		"APP_ENCRYPTION_KEY": base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))),
		"DATABASE_URL":       "postgres://seafarers:secret@postgres:5432/seafarers?sslmode=disable",
		"REDIS_URL":          "redis://redis:6379/0",
		"GOTENBERG_URL":      "http://gotenberg:3000",
		"REVALIDATE_SECRET":  strings.Repeat("r", 64),
		"LOG_LEVEL":          "debug",
	}
}

func getenv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadValidLocalConfig(t *testing.T) {
	cfg, err := Load(getenv(validEnv()))
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:5173", cfg.AdminURL)
	assert.Len(t, cfg.EncryptionKey, 32)
	assert.Equal(t, slog.LevelDebug, cfg.LogLevel)
	assert.True(t, cfg.IsLocal())
	assert.False(t, cfg.SecureCookies())
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	cases := []struct {
		name string
		key  string
		val  string
		msg  string
	}{
		{"unknown env", "APP_ENV", "dev", "APP_ENV"},
		{"placeholder encryption key", "APP_ENCRYPTION_KEY", "replace_with_32_byte_key", "APP_ENCRYPTION_KEY"},
		{"short encryption key", "APP_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString([]byte("short")), "APP_ENCRYPTION_KEY"},
		{"missing app url", "APP_URL", "", "APP_URL is required"},
		{"app url with path", "APP_URL", "http://localhost:3000/site", "APP_URL"},
		{"mysql database", "DATABASE_URL", "mysql://x@y/z", "DATABASE_URL"},
		{"missing redis", "REDIS_URL", "", "REDIS_URL is required"},
		{"missing gotenberg", "GOTENBERG_URL", "", "GOTENBERG_URL is required"},
		{"short revalidate secret", "REVALIDATE_SECRET", "replace_with_secure_secret", "REVALIDATE_SECRET"},
		{"bad log level", "LOG_LEVEL", "trace", "LOG_LEVEL"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := validEnv()
			env[tc.key] = tc.val
			_, err := Load(getenv(env))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.msg)
		})
	}
}

func TestLoadRequiresHTTPSOutsideLocal(t *testing.T) {
	env := validEnv()
	env["APP_ENV"] = "production"
	_, err := Load(getenv(env))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must use https outside local")

	env["APP_URL"] = "https://seafarerscrew.com"
	env["ADMIN_URL"] = "https://admin.seafarerscrew.com"
	env["API_URL"] = "https://api.seafarerscrew.com"
	cfg, err := Load(getenv(env))
	require.NoError(t, err)
	assert.True(t, cfg.SecureCookies())
}

func TestLoadReportsAllErrorsAtOnce(t *testing.T) {
	_, err := Load(getenv(map[string]string{}))
	require.Error(t, err)
	for _, name := range []string{"APP_ENV", "APP_URL", "ADMIN_URL", "API_URL", "APP_ENCRYPTION_KEY", "DATABASE_URL", "REDIS_URL", "GOTENBERG_URL", "REVALIDATE_SECRET"} {
		assert.Contains(t, err.Error(), name)
	}
}

func TestS3BootstrapConfigured(t *testing.T) {
	assert.False(t, S3Bootstrap{}.Configured())
	assert.True(t, S3Bootstrap{Bucket: "b", AccessKeyID: "a", SecretAccessKey: "s"}.Configured())
}
