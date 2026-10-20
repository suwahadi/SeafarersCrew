// Package app wires configuration, infrastructure, services and HTTP routes.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/suwahadi/seafarerscrew/backend/database/seeders"
	"github.com/suwahadi/seafarerscrew/backend/internal/config"
	"github.com/suwahadi/seafarerscrew/backend/internal/handler"
	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
	"github.com/suwahadi/seafarerscrew/backend/internal/job"
	"github.com/suwahadi/seafarerscrew/backend/internal/middleware"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/brevo"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/gotenberg"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/imaging"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/pdf"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/redis"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/s3"
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
	Config     config.Config
	Logger     *slog.Logger
	DB         *pgxpool.Pool
	Redis      *goredis.Client
	Auth       *service.AuthService
	Settings   *service.SettingsService
	Jobs       *job.Queue
	CPU        *job.Pool
	Storage    *s3.Storage
	Files      *service.FileService
	Crews      *service.CrewService
	CrewExport *service.CrewExportService
	CrewImport *service.CrewImportService
	Email      *service.EmailService
	brevo      *brevo.Client
	counter    *repository.LimiterRepository
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
	if err := imaging.Startup(logger); err != nil {
		db.Close()
		_ = rdb.Close()
		return nil, err
	}
	cpu := job.NewPool(0)
	storage := s3.New(storageConfig(settings))
	files := service.NewFileService(
		repository.NewFileRepository(db),
		storage,
		imaging.NewConverter(),
		pdf.NewProcessor("gs", 2*time.Minute),
		gotenberg.NewClient(cfg.GotenbergURL, 2*time.Minute),
		settings,
		cpu,
		time.Now,
	)
	a := &App{
		Config:   cfg,
		Logger:   logger,
		DB:       db,
		Redis:    rdb,
		Auth:     auth,
		Settings: settings,
		Jobs:     job.NewQueue(repository.NewJobRepository(db), rdb, logger),
		CPU:      cpu,
		Storage:  storage,
		Files:    files,
		counter:  counter,
	}
	a.wireCrew()
	a.wireEmail()
	return a, nil
}

func (a *App) wireCrew() {
	crewRepo := repository.NewCrewRepository(a.DB)
	a.Crews = service.NewCrewService(crewRepo, repository.NewCrewRecordRepository(a.DB), a.Settings, time.Now)
	a.CrewExport = service.NewCrewExportService(a.Crews, a.Jobs, a.Files, a.Settings)
	a.CrewImport = service.NewCrewImportService(crewRepo, repository.NewCacheRepository(a.Redis), a.Jobs)
	a.Jobs.Register(service.CrewExportJob, a.CrewExport.Handle)
	a.Jobs.Register(service.CrewImportJob, a.CrewImport.Handle)
}

func storageConfig(settings *service.SettingsService) s3.ConfigSource {
	return func(ctx context.Context) (s3.Config, error) {
		v, err := settings.Values(ctx, "storage")
		if err != nil {
			return s3.Config{}, err
		}
		str := func(k string) string {
			s, _ := v[k].(string)
			return s
		}
		pathStyle, _ := v["use_path_style"].(bool)
		minutes, _ := v["presign_minutes"].(int)
		return s3.Config{
			Provider:        str("provider"),
			Endpoint:        str("endpoint"),
			Region:          str("region"),
			Bucket:          str("bucket"),
			AccessKeyID:     str("access_key_id"),
			SecretAccessKey: str("secret_access_key"),
			UsePathStyle:    pathStyle,
			PresignMinutes:  minutes,
		}, nil
	}
}

// Close releases connections.
func (a *App) Close() {
	imaging.Shutdown()
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
	if err := seeders.Reference(ctx, a.DB, a.Settings, overrides); err != nil {
		return err
	}
	if overrides != nil {
		if err := a.Storage.EnsureBucket(ctx); err != nil {
			a.Logger.WarnContext(ctx, "storage bucket not ensured; configure Settings > Object Storage", "error", err)
		}
	}
	return nil
}

// Router builds the HTTP handler tree.
func (a *App) Router() http.Handler {
	masters := newMasterData(a.DB, a.Redis)
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
		"storage":  a.Storage,
	}))
	routes.Admin(r, routes.AdminDeps{
		Origin:        a.Config.AdminURL,
		Authenticator: a.Auth,
		Counter:       a.counter,
		Auth:          admin.NewAuthHandler(a.Auth, middleware.NewCookieWriter(a.Config.SecureCookies(), SessionAbsolute)),
		Settings:      admin.NewSettingsHandler(a.Settings),
		Jobs:          admin.NewJobHandler(a.Jobs),
		Crews:         admin.NewCrewHandler(a.Crews),
		CrewTransfer:  admin.NewCrewTransferHandler(a.CrewExport, a.CrewImport, a.Files),
		Masters:       admin.NewMasterRoutes(masters.services),
		MasterData:    admin.NewMasterDataHandler(masters.options, masters.compliance),
		Files:         admin.NewFileHandler(a.Files),
		Email:         a.emailHandler(),
		Contracts:     a.contractRoutes(),
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
	workerCtx, stopWorkers := context.WithCancel(ctx)
	defer stopWorkers()
	workersDone := make(chan struct{})
	go func() {
		a.Jobs.Run(workerCtx, max(2, runtime.NumCPU()/2))
		close(workersDone)
	}()
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
	err := srv.Shutdown(shutdownCtx)
	stopWorkers()
	<-workersDone
	if err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	return nil
}
