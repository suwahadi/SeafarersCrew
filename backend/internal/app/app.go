// Package app wires configuration, infrastructure, services and HTTP routes.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/suwahadi/seafarerscrew/backend/database/seeders"
	"github.com/suwahadi/seafarerscrew/backend/internal/config"
	"github.com/suwahadi/seafarerscrew/backend/internal/handler"
	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
	"github.com/suwahadi/seafarerscrew/backend/internal/middleware"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/redis"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/routes"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/password"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/secret"
)

// Session lifetimes from PRD Section 5.
const (
	SessionIdleTimeout = 8 * time.Hour
	SessionAbsolute    = 7 * 24 * time.Hour
	requestTimeout     = 30 * time.Second
)

// App holds long-lived dependencies.
type App struct {
	Config   config.Config
	Logger   *slog.Logger
	DB       *pgxpool.Pool
	Redis    *goredis.Client
	Auth     *service.AuthService
	Settings *service.SettingsService
	counter  *repository.LimiterRepository
}

// NewLogger builds the JSON logger for level.
func NewLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

// New connects to PostgreSQL and Redis and builds the services.
func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	rdb, err := redis.Open(ctx, cfg.RedisURL)
	if err != nil {
		db.Close()
		return nil, err
	}
	box, err := secret.NewBox(cfg.EncryptionKey)
	if err != nil {
		db.Close()
		_ = rdb.Close()
		return nil, fmt.Errorf("create secret box: %w", err)
	}
	counter := repository.NewLimiterRepository(rdb)
	auth, err := service.NewAuthService(
		repository.NewAdminRepository(db),
		repository.NewSessionRepository(rdb, SessionIdleTimeout, SessionAbsolute, time.Now),
		counter,
		password.NewHasher(password.DefaultParams),
		time.Now,
	)
	if err != nil {
		db.Close()
		_ = rdb.Close()
		return nil, err
	}
	settings := service.NewSettingsService(repository.NewSettingRepository(db), repository.NewCacheRepository(rdb), box, nil)
	return &App{Config: cfg, Logger: logger, DB: db, Redis: rdb, Auth: auth, Settings: settings, counter: counter}, nil
}

// Close releases connections.
func (a *App) Close() {
	a.DB.Close()
	_ = a.Redis.Close()
}

// Seed loads reference data and copies S3 bootstrap values into Settings on first boot.
func (a *App) Seed(ctx context.Context) error {
	var overrides map[string]map[string]any
	if s := a.Config.S3Bootstrap; s.Configured() {
		storage := map[string]any{
			"bucket":            s.Bucket,
			"access_key_id":     s.AccessKeyID,
			"secret_access_key": s.SecretAccessKey,
			"use_path_style":    s.UsePathStyle || s.Endpoint != "",
		}
		if s.Endpoint != "" {
			storage["endpoint"] = s.Endpoint
		}
		if s.Region != "" {
			storage["region"] = s.Region
		}
		overrides = map[string]map[string]any{"storage": storage}
	}
	return seeders.Reference(ctx, a.DB, a.Settings, overrides)
}

// Router builds the HTTP handler tree.
func (a *App) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID(a.Logger))
	r.Use(middleware.RealIP)
	r.Use(middleware.AccessLog)
	r.Use(middleware.Recover)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.Timeout(requestTimeout))

	routes.Health(r, handler.NewHealthHandler(map[string]handler.Pinger{
		"database": handler.PingFunc(a.DB.Ping),
		"redis":    handler.PingFunc(func(ctx context.Context) error { return a.Redis.Ping(ctx).Err() }),
	}))
	routes.Admin(r, routes.AdminDeps{
		Origin:        a.Config.AdminURL,
		Authenticator: a.Auth,
		Counter:       a.counter,
		Auth:          admin.NewAuthHandler(a.Auth, middleware.NewCookieWriter(a.Config.SecureCookies(), SessionAbsolute)),
		Settings:      admin.NewSettingsHandler(a.Settings),
	})
	routes.Public(r, routes.PublicDeps{Origin: a.Config.AppURL, Counter: a.counter})
	return r
}

// Serve runs the HTTP server until ctx is cancelled, then shuts down gracefully.
func (a *App) Serve(ctx context.Context) error {
	srv := &http.Server{
		Addr:              a.Config.HTTPAddr,
		Handler:           a.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 16,
	}
	errCh := make(chan error, 1)
	go func() {
		a.Logger.Info("http server listening", "addr", srv.Addr, "env", a.Config.Env)
		errCh <- srv.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	a.Logger.Info("http server shutting down")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	return nil
}
