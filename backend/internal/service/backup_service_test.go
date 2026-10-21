package service

import (
	"context"
	"errors"
	"io"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/job"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

type memBackups struct{ rows map[uuid.UUID]*model.Backup }

func newMemBackups() *memBackups { return &memBackups{rows: map[uuid.UUID]*model.Backup{}} }

func (m *memBackups) add(status string, started time.Time, finished *time.Time, key string) uuid.UUID {
	id := uuid.New()
	m.rows[id] = &model.Backup{ID: id, Status: status, StartedAt: started, FinishedAt: finished, StorageKey: key}
	return id
}

func (m *memBackups) Create(_ context.Context, at time.Time) (model.Backup, error) {
	id := m.add(model.BackupRunning, at, nil, "")
	return *m.rows[id], nil
}

func (m *memBackups) Get(_ context.Context, id uuid.UUID) (model.Backup, error) {
	if b, ok := m.rows[id]; ok {
		return *b, nil
	}
	return model.Backup{}, repository.ErrNotFound
}

func (m *memBackups) Finish(_ context.Context, id uuid.UUID, status, key string, size int64, errText string, at time.Time) error {
	b := m.rows[id]
	b.Status, b.StorageKey, b.SizeBytes, b.Error, b.FinishedAt = status, key, size, errText, &at
	return nil
}

func (m *memBackups) List(context.Context, int, int) (repository.Page[model.Backup], error) {
	return repository.Page[model.Backup]{}, nil
}

func (m *memBackups) Since(_ context.Context, t time.Time) ([]model.Backup, error) {
	var out []model.Backup
	for _, b := range m.rows {
		if !b.StartedAt.Before(t) {
			out = append(out, *b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	return out, nil
}

func (m *memBackups) FailStale(_ context.Context, cutoff, at time.Time) (int64, error) {
	var n int64
	for _, b := range m.rows {
		if b.Status == model.BackupRunning && b.StartedAt.Before(cutoff) {
			b.Status, b.FinishedAt = model.BackupFailed, &at
			n++
		}
	}
	return n, nil
}

func (m *memBackups) Expired(_ context.Context, cutoff time.Time) ([]model.Backup, error) {
	var out []model.Backup
	for _, b := range m.rows {
		if b.StartedAt.Before(cutoff) && b.Status != model.BackupRunning {
			out = append(out, *b)
		}
	}
	return out, nil
}

func (m *memBackups) Delete(_ context.Context, ids []uuid.UUID) error {
	for _, id := range ids {
		delete(m.rows, id)
	}
	return nil
}

type fakeDumper struct {
	data []byte
	err  error
	urls []string
}

func (f *fakeDumper) Dump(_ context.Context, url string, w io.Writer) error {
	f.urls = append(f.urls, url)
	if f.err != nil {
		return f.err
	}
	_, err := w.Write(f.data)
	return err
}

type memBackupStorage struct {
	objects map[string][]byte
	putErr  error
	deleted []string
}

func (m *memBackupStorage) PutReader(_ context.Context, key string, r io.Reader, size int64, _ string) error {
	if m.putErr != nil {
		return m.putErr
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if int64(len(b)) != size {
		return errors.New("size mismatch")
	}
	m.objects[key] = b
	return nil
}

func (m *memBackupStorage) Delete(_ context.Context, keys ...string) error {
	for _, k := range keys {
		delete(m.objects, k)
		m.deleted = append(m.deleted, k)
	}
	return nil
}

type inlinePool struct{}

func (inlinePool) Do(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

type backupFixture struct {
	svc     *BackupService
	store   *memBackups
	dumper  *fakeDumper
	storage *memBackupStorage
	queue   *fakeQueue
	now     time.Time
}

func newBackupFixture(t *testing.T, now time.Time, backup map[string]any) *backupFixture {
	t.Helper()
	f := &backupFixture{store: newMemBackups(), dumper: &fakeDumper{data: []byte("PGDMP")}, storage: &memBackupStorage{objects: map[string][]byte{}}, queue: &fakeQueue{}, now: now}
	settings := fakeSettings{"general": {"timezone": "Asia/Jakarta"}, "alerts": {}, "backup": backup}
	f.svc = NewBackupService(f.store, f.dumper, f.storage, settings, f.queue, inlinePool{}, "postgres://u:p@db/seafarers", t.TempDir(), func() time.Time { return f.now })
	return f
}

func defaultBackupSettings() map[string]any {
	return map[string]any{"enabled": true, "run_time": "02:00", "retention_days": 30, "prefix": "backups/"}
}

func wib(t *testing.T, y int, m time.Month, d, h, min int) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Jakarta")
	require.NoError(t, err)
	return time.Date(y, m, d, h, min, 0, 0, loc).UTC()
}

func TestExecuteUploadsTheDumpAndRecordsSuccess(t *testing.T) {
	f := newBackupFixture(t, wib(t, 2026, 10, 21, 2, 5), defaultBackupSettings())
	b, err := f.store.Create(context.Background(), f.now)
	require.NoError(t, err)

	done, err := f.svc.Execute(context.Background(), b.ID)

	require.NoError(t, err)
	assert.Equal(t, model.BackupSucceeded, done.Status)
	assert.Equal(t, int64(5), done.SizeBytes)
	assert.Equal(t, "backups/seafarers-20261020-190500.dump", done.StorageKey)
	assert.Equal(t, []byte("PGDMP"), f.storage.objects[done.StorageKey])
	assert.Equal(t, []string{"postgres://u:p@db/seafarers"}, f.dumper.urls)
}

func TestExecuteRecordsFailureWhenTheDumpFails(t *testing.T) {
	f := newBackupFixture(t, wib(t, 2026, 10, 21, 2, 5), defaultBackupSettings())
	f.dumper.err = errors.New("pg_dump: connection refused")
	b, _ := f.store.Create(context.Background(), f.now)

	_, err := f.svc.Execute(context.Background(), b.ID)

	require.Error(t, err)
	got, _ := f.store.Get(context.Background(), b.ID)
	assert.Equal(t, model.BackupFailed, got.Status)
	assert.Contains(t, got.Error, "connection refused")
	assert.Empty(t, f.storage.objects)
}

func TestExecuteRecordsFailureWhenTheUploadFailsOrTheDumpIsEmpty(t *testing.T) {
	f := newBackupFixture(t, wib(t, 2026, 10, 21, 2, 5), defaultBackupSettings())
	f.storage.putErr = errors.New("bucket unreachable")
	b, _ := f.store.Create(context.Background(), f.now)
	_, err := f.svc.Execute(context.Background(), b.ID)
	require.Error(t, err)
	got, _ := f.store.Get(context.Background(), b.ID)
	assert.Equal(t, model.BackupFailed, got.Status)

	f.storage.putErr = nil
	f.dumper.data = nil
	b, _ = f.store.Create(context.Background(), f.now)
	_, err = f.svc.Execute(context.Background(), b.ID)
	assert.ErrorContains(t, err, "empty")
}

func TestBackupPrefixMustStayPrivate(t *testing.T) {
	cases := map[string]bool{"backups/": true, "ops/db": true, "": true, "public/backups/": false, "../x/": false, "/abs/": false, "a//b/": false}
	for prefix, ok := range cases {
		_, err := NormalizeBackupPrefix(prefix)
		assert.Equal(t, ok, err == nil, "prefix %q", prefix)
	}
	f := newBackupFixture(t, wib(t, 2026, 10, 21, 3, 0), map[string]any{"enabled": true, "run_time": "02:00", "retention_days": 30, "prefix": "public/backups/"})
	_, _, err := f.svc.Start(context.Background())
	assert.Error(t, err)
	assert.Empty(t, f.store.rows)
}

func TestExecutePrunesBackupsPastRetention(t *testing.T) {
	f := newBackupFixture(t, wib(t, 2026, 10, 21, 2, 5), defaultBackupSettings())
	old := f.now.AddDate(0, 0, -45)
	oldID := f.store.add(model.BackupSucceeded, old, &old, "backups/old.dump")
	f.storage.objects["backups/old.dump"] = []byte("x")
	recent := f.now.AddDate(0, 0, -5)
	recentID := f.store.add(model.BackupSucceeded, recent, &recent, "backups/recent.dump")
	b, _ := f.store.Create(context.Background(), f.now)

	_, err := f.svc.Execute(context.Background(), b.ID)

	require.NoError(t, err)
	assert.NotContains(t, f.store.rows, oldID)
	assert.Contains(t, f.store.rows, recentID)
	assert.Equal(t, []string{"backups/old.dump"}, f.storage.deleted)
}

func TestScheduleIfDueWaitsForTheConfiguredTime(t *testing.T) {
	f := newBackupFixture(t, wib(t, 2026, 10, 21, 1, 59), defaultBackupSettings())
	started, err := f.svc.ScheduleIfDue(context.Background())
	require.NoError(t, err)
	assert.False(t, started)

	f.now = wib(t, 2026, 10, 21, 2, 0)
	started, err = f.svc.ScheduleIfDue(context.Background())
	require.NoError(t, err)
	assert.True(t, started)
	assert.Equal(t, []string{BackupJob}, f.queue.kinds)
	assert.Len(t, f.store.rows, 1)
}

func TestScheduleIfDueRunsOncePerDay(t *testing.T) {
	f := newBackupFixture(t, wib(t, 2026, 10, 21, 2, 0), defaultBackupSettings())
	for range 3 {
		_, err := f.svc.ScheduleIfDue(context.Background())
		require.NoError(t, err)
		f.now = f.now.Add(time.Minute)
	}
	assert.Len(t, f.store.rows, 1, "a running backup blocks another start")

	for _, b := range f.store.rows {
		done := f.now
		b.Status, b.FinishedAt = model.BackupSucceeded, &done
	}
	f.now = wib(t, 2026, 10, 21, 18, 0)
	started, err := f.svc.ScheduleIfDue(context.Background())
	require.NoError(t, err)
	assert.False(t, started, "a successful backup today blocks another start")

	f.now = wib(t, 2026, 10, 22, 2, 1)
	started, err = f.svc.ScheduleIfDue(context.Background())
	require.NoError(t, err)
	assert.True(t, started, "the next day schedules again")
}

func TestScheduleIfDueRetriesAFailedBackupAfterAnHourUpToALimit(t *testing.T) {
	f := newBackupFixture(t, wib(t, 2026, 10, 21, 2, 0), defaultBackupSettings())
	failedAt := f.now
	f.store.add(model.BackupFailed, f.now, &failedAt, "")

	f.now = failedAt.Add(30 * time.Minute)
	started, _ := f.svc.ScheduleIfDue(context.Background())
	assert.False(t, started, "too soon to retry")

	f.now = failedAt.Add(61 * time.Minute)
	started, err := f.svc.ScheduleIfDue(context.Background())
	require.NoError(t, err)
	assert.True(t, started)

	capped := newBackupFixture(t, wib(t, 2026, 10, 21, 12, 0), defaultBackupSettings())
	for i := range backupMaxRetries {
		at := wib(t, 2026, 10, 21, 2, i)
		capped.store.add(model.BackupFailed, at, &at, "")
	}
	started, _ = capped.svc.ScheduleIfDue(context.Background())
	assert.False(t, started, "too many failures today")
}

func TestScheduleIfDueFailsBackupsStuckRunning(t *testing.T) {
	f := newBackupFixture(t, wib(t, 2026, 10, 21, 8, 0), defaultBackupSettings())
	stuck := f.store.add(model.BackupRunning, wib(t, 2026, 10, 21, 2, 0), nil, "")

	_, err := f.svc.ScheduleIfDue(context.Background())

	require.NoError(t, err)
	assert.Equal(t, model.BackupFailed, f.store.rows[stuck].Status)
}

func TestScheduleIfDueDoesNothingWhenDisabled(t *testing.T) {
	cfg := defaultBackupSettings()
	cfg["enabled"] = false
	f := newBackupFixture(t, wib(t, 2026, 10, 21, 12, 0), cfg)
	started, err := f.svc.ScheduleIfDue(context.Background())
	require.NoError(t, err)
	assert.False(t, started)
	assert.Empty(t, f.store.rows)
}

func TestHandleTurnsBadPayloadsIntoPermanentFailures(t *testing.T) {
	f := newBackupFixture(t, wib(t, 2026, 10, 21, 2, 5), defaultBackupSettings())
	_, err := f.svc.Handle(context.Background(), model.Job{Payload: []byte("{")})
	var perm job.Permanent
	assert.True(t, errors.As(err, &perm))
}
