// Package config loads typed configuration from environment variables.
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
)

// Environment names accepted in APP_ENV.
const (
	EnvLocal      = "local"
	EnvStaging    = "staging"
	EnvProduction = "production"
)

// S3Bootstrap holds optional initial storage values copied into Settings on first boot.
type S3Bootstrap struct {
	Endpoint        string
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
}

// Configured reports whether enough values exist to seed storage settings.
func (s S3Bootstrap) Configured() bool {
	return s.Bucket != "" && s.AccessKeyID != "" && s.SecretAccessKey != ""
}

// Config is the bootstrap configuration of the API process.
type Config struct {
	Env              string
	AppURL           string
	AdminURL         string
	APIURL           string
	EncryptionKey    []byte
	DatabaseURL      string
	RedisURL         string
	GotenbergURL     string
	RevalidateSecret string
	WebURL           string
	S3Bootstrap      S3Bootstrap
	LogLevel         slog.Level
	HTTPAddr         string
}

// IsLocal reports whether the process runs in the local environment.
func (c Config) IsLocal() bool { return c.Env == EnvLocal }

// SecureCookies reports whether cookies must carry the Secure flag.
func (c Config) SecureCookies() bool {
	return !c.IsLocal() || strings.HasPrefix(c.AdminURL, "https://")
}

// Load reads configuration using getenv, typically os.Getenv.
func Load(getenv func(string) string) (Config, error) {
	var errs []error
	cfg := Config{
		Env:              strings.TrimSpace(getenv("APP_ENV")),
		DatabaseURL:      strings.TrimSpace(getenv("DATABASE_URL")),
		RedisURL:         strings.TrimSpace(getenv("REDIS_URL")),
		GotenbergURL:     strings.TrimSpace(getenv("GOTENBERG_URL")),
		RevalidateSecret: strings.TrimSpace(getenv("REVALIDATE_SECRET")),
		WebURL:           strings.TrimSpace(getenv("WEB_URL")),
		HTTPAddr:         ":8080",
		S3Bootstrap: S3Bootstrap{
			Endpoint:        strings.TrimSpace(getenv("S3_BOOTSTRAP_ENDPOINT")),
			Bucket:          strings.TrimSpace(getenv("S3_BOOTSTRAP_BUCKET")),
			Region:          strings.TrimSpace(getenv("S3_BOOTSTRAP_REGION")),
			AccessKeyID:     strings.TrimSpace(getenv("S3_BOOTSTRAP_ACCESS_KEY_ID")),
			SecretAccessKey: strings.TrimSpace(getenv("S3_BOOTSTRAP_SECRET_ACCESS_KEY")),
			UsePathStyle:    strings.EqualFold(strings.TrimSpace(getenv("S3_BOOTSTRAP_USE_PATH_STYLE")), "true"),
		},
	}

	switch cfg.Env {
	case EnvLocal, EnvStaging, EnvProduction:
	default:
		errs = append(errs, fmt.Errorf("APP_ENV must be one of local, staging, production; got %q", cfg.Env))
	}

	if addr := strings.TrimSpace(getenv("HTTP_ADDR")); addr != "" {
		cfg.HTTPAddr = addr
	}

	var err error
	if cfg.AppURL, err = origin("APP_URL", getenv("APP_URL")); err != nil {
		errs = append(errs, err)
	}
	if cfg.AdminURL, err = origin("ADMIN_URL", getenv("ADMIN_URL")); err != nil {
		errs = append(errs, err)
	}
	if cfg.APIURL, err = origin("API_URL", getenv("API_URL")); err != nil {
		errs = append(errs, err)
	}
	cfg.WebURL = strings.TrimRight(cfg.WebURL, "/")
	if cfg.WebURL == "" {
		cfg.WebURL = cfg.AppURL
	} else if cfg.WebURL, err = origin("WEB_URL", cfg.WebURL); err != nil {
		errs = append(errs, err)
	}
	if cfg.EncryptionKey, err = encryptionKey(getenv("APP_ENCRYPTION_KEY")); err != nil {
		errs = append(errs, err)
	}
	if err = requireScheme("DATABASE_URL", cfg.DatabaseURL, "postgres", "postgresql"); err != nil {
		errs = append(errs, err)
	}
	if err = requireScheme("REDIS_URL", cfg.RedisURL, "redis", "rediss"); err != nil {
		errs = append(errs, err)
	}
	if err = requireScheme("GOTENBERG_URL", cfg.GotenbergURL, "http", "https"); err != nil {
		errs = append(errs, err)
	}
	if len(cfg.RevalidateSecret) < 32 {
		errs = append(errs, errors.New("REVALIDATE_SECRET must be at least 32 characters"))
	}
	if cfg.LogLevel, err = logLevel(getenv("LOG_LEVEL")); err != nil {
		errs = append(errs, err)
	}
	if cfg.Env != EnvLocal {
		for name, v := range map[string]string{"APP_URL": cfg.AppURL, "ADMIN_URL": cfg.AdminURL, "API_URL": cfg.APIURL} {
			if v != "" && !strings.HasPrefix(v, "https://") {
				errs = append(errs, fmt.Errorf("%s must use https outside local", name))
			}
		}
	}

	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %w", errors.Join(errs...))
	}
	return cfg, nil
}

func origin(name, raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || (u.Path != "" && u.Path != "/") || u.RawQuery != "" {
		return "", fmt.Errorf("%s must be an origin such as https://example.com", name)
	}
	return u.Scheme + "://" + u.Host, nil
}

func encryptionKey(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("APP_ENCRYPTION_KEY is required; generate one with: openssl rand -base64 32")
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(key) != 32 {
		return nil, errors.New("APP_ENCRYPTION_KEY must be base64 of exactly 32 bytes; generate one with: openssl rand -base64 32")
	}
	return key, nil
}

func requireScheme(name, raw string, schemes ...string) error {
	if raw == "" {
		return fmt.Errorf("%s is required", name)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%s is not a valid URL", name)
	}
	for _, s := range schemes {
		if u.Scheme == s {
			return nil
		}
	}
	return fmt.Errorf("%s must use scheme %s", name, strings.Join(schemes, " or "))
}

func logLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "", "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("LOG_LEVEL must be debug, info, warn or error; got %q", raw)
	}
}
