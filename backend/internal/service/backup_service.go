package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/job"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/logctx"
)

// BackupJob is the job kind that runs a queued backup.
const BackupJob = "backup.run"

const (
	backupStaleAfter   = 3 * time.Hour
	backupRetryAfter   = time.Hour
	backupMaxRetries   = 6
	backupErrorMaxSize = 500
)

// BackupStore persists backup history.
type BackupStore interface {
	Create(ctx context.Context, startedAt time.Time) (model.Backup, error)
	Get(ctx context.Context, id uuid.UUID) (model.Backup, error)
	Finish(ctx context.Context, id uuid.UUID, status, storageKey string, size int64, errText string, at time.Time) error
	List(ctx context.Context, page, perPage int) (repository.Page[model.Backup], error)
	Since(ctx context.Context, t time.Time) ([]model.Backup, error)
	FailStale(ctx context.Context, cutoff, at time.Time) (int64, error)
	Expired(ctx context.Context, cutoff time.Time) ([]model.Backup, error)
	Delete(ctx context.Context, ids []uuid.UUID) error
}

// Dumper writes a database dump.
type Dumper interface {
	Dump(ctx context.Context, databaseURL string, w io.Writer) error
}

// BackupStorage keeps backup files in private object storage.
type BackupStorage interface {
	PutReader(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, keys ...string) error
}

// BackupService creates, schedules and prunes database backups (PRD 14.7).
type BackupService struct {
	store       BackupStore
	dumper      Dumper
	storage     BackupStorage
	settings    SettingsReader
	jobs        Enqueuer
	pool        Bounded
	databaseURL string
	tempDir     string
	now         func() time.Time
}

// NewBackupService builds a BackupService; an empty tempDir uses the system temporary directory.
func NewBackupService(store BackupStore, dumper Dumper, storage BackupStorage, settings SettingsReader, jobs Enqueuer, pool Bounded, databaseURL, tempDir string, now func() time.Time) *BackupService {
	return &BackupService{store: store, dumper: dumper, storage: storage, settings: settings, jobs: jobs, pool: pool, databaseURL: databaseURL, tempDir: tempDir, now: now}
}

type backupPayload struct {
	BackupID uuid.UUID `json:"backup_id"`
}

type backupSettings struct {
	enabled   bool
	runTime   string
	retention int
	prefix    string
}

func (s *BackupService) loadSettings(ctx context.Context) (backupSettings, error) {
	v, err := s.settings.Values(ctx, "backup")
	if err != nil {
		return backupSettings{}, err
	}
	out := backupSettings{runTime: "02:00", retention: 30, prefix: "backups/"}
	if b, ok := v["enabled"].(bool); ok {
		out.enabled = b
	}
	if t, ok := v["run_time"].(string); ok && t != "" {
		out.runTime = t
	}
	if n, ok := v["retention_days"].(int); ok && n > 0 {
		out.retention = n
	}
	if p, ok := v["prefix"].(string); ok {
		out.prefix = p
	}
	prefix, err := NormalizeBackupPrefix(out.prefix)
	if err != nil {
		return backupSettings{}, err
	}
	out.prefix = prefix
	return out, nil
}

// NormalizeBackupPrefix returns prefix with a trailing slash, rejecting locations that are public or escape the bucket root.
func NormalizeBackupPrefix(prefix string) (string, error) {
	p := strings.TrimSpace(prefix)
	if p == "" {
		p = "backups/"
	}
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	switch {
	case strings.HasPrefix(p, "/"), strings.Contains(p, ".."), strings.Contains(p, "//"):
		return "", apperr.Validation(map[string]string{"prefix": "Use a plain folder path such as backups/."})
	case strings.HasPrefix(p, MediaPrefix):
		return "", apperr.Validation(map[string]string{"prefix": "Backups cannot be stored in the public folder."})
	}
	return p, nil
}

// Start records a running backup and queues the job that performs it.
func (s *BackupService) Start(ctx context.Context) (model.Backup, model.Job, error) {
	if _, err := s.loadSettings(ctx); err != nil {
		return model.Backup{}, model.Job{}, err
	}
	b, err := s.store.Create(ctx, s.now().UTC())
	if err != nil {
		return b, model.Job{}, err
	}
	j, err := s.jobs.Enqueue(ctx, BackupJob, backupPayload{BackupID: b.ID})
	if err != nil {
		s.fail(ctx, b.ID, err)
		return b, j, fmt.Errorf("queue backup: %w", err)
	}
	return b, j, nil
}

// Handle runs a queued backup.
func (s *BackupService) Handle(ctx context.Context, j model.Job) (any, error) {
	var p backupPayload
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return nil, job.Permanent{Err: fmt.Errorf("decode backup payload: %w", err)}
	}
	b, err := s.Execute(ctx, p.BackupID)
	if err != nil {
		return nil, job.Permanent{Err: err}
	}
	return map[string]any{"backup_id": b.ID, "storage_key": b.StorageKey, "size_bytes": b.SizeBytes}, nil
}

// RunNow performs a backup in the calling process, for the command line.
func (s *BackupService) RunNow(ctx context.Context) (model.Backup, error) {
	if _, err := s.loadSettings(ctx); err != nil {
		return model.Backup{}, err
	}
	b, err := s.store.Create(ctx, s.now().UTC())
	if err != nil {
		return b, err
	}
	return s.Execute(ctx, b.ID)
}

// Execute dumps the database, uploads the dump, records the outcome and prunes old backups.
func (s *BackupService) Execute(ctx context.Context, id uuid.UUID) (model.Backup, error) {
	cfg, err := s.loadSettings(ctx)
	if err != nil {
		s.fail(ctx, id, err)
		return model.Backup{}, err
	}
	key, size, err := s.dumpAndUpload(ctx, cfg.prefix)
	if err != nil {
		s.fail(ctx, id, err)
		return model.Backup{}, err
	}
	if err := s.store.Finish(ctx, id, model.BackupSucceeded, key, size, "", s.now().UTC()); err != nil {
		return model.Backup{}, err
	}
	if err := s.Prune(ctx); err != nil {
		logctx.From(ctx).WarnContext(ctx, "backup retention cleanup failed", "error", err)
	}
	return s.store.Get(ctx, id)
}

func (s *BackupService) dumpAndUpload(ctx context.Context, prefix string) (string, int64, error) {
	f, err := os.CreateTemp(s.tempDir, "seafarers-backup-*.dump")
	if err != nil {
		return "", 0, fmt.Errorf("create temporary backup file: %w", err)
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()
	err = s.pool.Do(ctx, func(ctx context.Context) error { return s.dumper.Dump(ctx, s.databaseURL, f) })
	if err != nil {
		return "", 0, err
	}
	info, err := f.Stat()
	if err != nil {
		return "", 0, fmt.Errorf("stat backup file: %w", err)
	}
	if info.Size() == 0 {
		return "", 0, errors.New("the database dump is empty")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", 0, fmt.Errorf("rewind backup file: %w", err)
	}
	key := prefix + "seafarers-" + s.now().UTC().Format("20060102-150405") + ".dump"
	if err := s.storage.PutReader(ctx, key, f, info.Size(), "application/octet-stream"); err != nil {
		return "", 0, err
	}
	return key, info.Size(), nil
}

func (s *BackupService) fail(ctx context.Context, id uuid.UUID, cause error) {
	msg := cause.Error()
	if len(msg) > backupErrorMaxSize {
		msg = msg[:backupErrorMaxSize]
	}
	if err := s.store.Finish(context.WithoutCancel(ctx), id, model.BackupFailed, "", 0, msg, s.now().UTC()); err != nil {
		logctx.From(ctx).ErrorContext(ctx, "backup failure not recorded", "error", err)
	}
}

// Prune deletes backups, and their files, older than the retention period.
func (s *BackupService) Prune(ctx context.Context) error {
	cfg, err := s.loadSettings(ctx)
	if err != nil {
		return err
	}
	old, err := s.store.Expired(ctx, s.now().UTC().AddDate(0, 0, -cfg.retention))
	if err != nil || len(old) == 0 {
		return err
	}
	ids := make([]uuid.UUID, 0, len(old))
	var keys []string
	for _, b := range old {
		ids = append(ids, b.ID)
		if b.StorageKey != "" {
			keys = append(keys, b.StorageKey)
		}
	}
	if err := s.storage.Delete(ctx, keys...); err != nil {
		return err
	}
	return s.store.Delete(ctx, ids)
}

// ScheduleIfDue starts the daily backup once the configured time has passed and none ran since.
func (s *BackupService) ScheduleIfDue(ctx context.Context) (bool, error) {
	cfg, err := s.loadSettings(ctx)
	if err != nil || !cfg.enabled {
		return false, err
	}
	loc, _, err := LoadDisplaySettings(ctx, s.settings)
	if err != nil {
		return false, err
	}
	now := s.now().UTC()
	scheduled, err := scheduledAt(now, loc, cfg.runTime)
	if err != nil || now.Before(scheduled) {
		return false, err
	}
	if _, err := s.store.FailStale(ctx, now.Add(-backupStaleAfter), now); err != nil {
		return false, err
	}
	recent, err := s.store.Since(ctx, scheduled)
	if err != nil {
		return false, err
	}
	failures := 0
	for _, b := range recent {
		switch {
		case b.Status == model.BackupRunning || b.Status == model.BackupSucceeded:
			return false, nil
		case b.FinishedAt != nil && now.Sub(*b.FinishedAt) < backupRetryAfter:
			return false, nil
		default:
			failures++
		}
	}
	if failures >= backupMaxRetries {
		return false, nil
	}
	if _, _, err := s.Start(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func scheduledAt(now time.Time, loc *time.Location, runTime string) (time.Time, error) {
	t, err := time.Parse("15:04", runTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse backup run time %q: %w", runTime, err)
	}
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), t.Hour(), t.Minute(), 0, 0, loc).UTC(), nil
}

// List returns the backup history, newest first.
func (s *BackupService) List(ctx context.Context, page, perPage int) (repository.Page[model.Backup], error) {
	return s.store.List(ctx, page, perPage)
}
