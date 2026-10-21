package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

type fakeMaintenance struct {
	closedToday   time.Time
	closed        int64
	newsFrom      time.Time
	newsTo        time.Time
	newsCount     int
	batches       [][]model.File
	purgeCutoff   time.Time
	purgedIDs     []uuid.UUID
	purgeCalls    int
	purgeErr      error
	applicants    [][]model.ExpiredApplicant
	applicantCall int
	appCutoff     time.Time
	deletedApps   []uuid.UUID
	deletedFiles  []uuid.UUID
	markPurgedErr error
}

func (f *fakeMaintenance) CloseExpiredVacancies(_ context.Context, today time.Time) (int64, error) {
	f.closedToday = today
	return f.closed, nil
}

func (f *fakeMaintenance) CountNewlyLiveNews(_ context.Context, from, to time.Time) (int, error) {
	f.newsFrom, f.newsTo = from, to
	return f.newsCount, nil
}

func (f *fakeMaintenance) PurgeCandidates(_ context.Context, cutoff time.Time, _ int) ([]model.File, error) {
	f.purgeCutoff = cutoff
	if f.purgeErr != nil {
		return nil, f.purgeErr
	}
	if f.purgeCalls >= len(f.batches) {
		return nil, nil
	}
	f.purgeCalls++
	return f.batches[f.purgeCalls-1], nil
}

func (f *fakeMaintenance) ExpiredApplicants(_ context.Context, cutoff time.Time, _ int) ([]model.ExpiredApplicant, error) {
	f.appCutoff = cutoff
	if f.applicantCall >= len(f.applicants) {
		return nil, nil
	}
	f.applicantCall++
	return f.applicants[f.applicantCall-1], nil
}

func (f *fakeMaintenance) DeleteApplicants(_ context.Context, ids, fileIDs []uuid.UUID, _ time.Time) error {
	f.deletedApps = append(f.deletedApps, ids...)
	f.deletedFiles = append(f.deletedFiles, fileIDs...)
	return nil
}

func (f *fakeMaintenance) MarkPurged(_ context.Context, ids []uuid.UUID, _ time.Time) error {
	f.purgedIDs = append(f.purgedIDs, ids...)
	return f.markPurgedErr
}

type fakeExpirer struct {
	n     int64
	today time.Time
}

func (f *fakeExpirer) ExpireQuotations(_ context.Context, today time.Time) (int64, error) {
	f.today = today
	return f.n, nil
}

type recordingSite struct{ tags [][]string }

func (r *recordingSite) Changed(_ context.Context, tags ...string) error {
	r.tags = append(r.tags, tags)
	return nil
}

type memCursor struct{ data map[string][]byte }

func (m *memCursor) Get(_ context.Context, key string) ([]byte, bool, error) {
	v, ok := m.data[key]
	return v, ok, nil
}

func (m *memCursor) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.data[key] = value
	return nil
}

type maintenanceFixture struct {
	svc        *MaintenanceService
	store      *fakeMaintenance
	quotations *fakeExpirer
	dashboard  *countingInvalidator
	site       *recordingSite
	cursor     *memCursor
	storage    *memBackupStorage
	now        time.Time
}

func newMaintenanceFixture(t *testing.T, now time.Time) *maintenanceFixture {
	t.Helper()
	f := &maintenanceFixture{store: &fakeMaintenance{}, quotations: &fakeExpirer{}, dashboard: &countingInvalidator{}, site: &recordingSite{}, cursor: &memCursor{data: map[string][]byte{}}, storage: &memBackupStorage{objects: map[string][]byte{}}, now: now}
	settings := fakeSettings{"general": {"timezone": "Asia/Jakarta"}, "alerts": {}}
	f.svc = NewMaintenanceService(f.store, f.quotations, f.dashboard, f.site, f.cursor, f.storage, settings, func() time.Time { return f.now })
	return f
}

func TestCloseVacanciesUsesTheLocalCalendarDateAndRefreshesTheSite(t *testing.T) {
	f := newMaintenanceFixture(t, time.Date(2026, 10, 20, 18, 30, 0, 0, time.UTC))
	f.store.closed = 2

	require.NoError(t, f.svc.CloseVacancies(context.Background()))

	assert.Equal(t, time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC), f.store.closedToday, "01:30 WIB on the 21st is already the 21st")
	assert.Equal(t, [][]string{{"vacancies", "home", "sitemap"}}, f.site.tags)
	assert.Equal(t, 1, f.dashboard.calls)
}

func TestCloseVacanciesDoesNotTouchTheSiteWhenNothingClosed(t *testing.T) {
	f := newMaintenanceFixture(t, time.Date(2026, 10, 21, 5, 0, 0, 0, time.UTC))
	require.NoError(t, f.svc.CloseVacancies(context.Background()))
	assert.Empty(t, f.site.tags)
	assert.Zero(t, f.dashboard.calls)
}

func TestExpireQuotationsRefreshesTheDashboardOnlyWhenSomeExpired(t *testing.T) {
	f := newMaintenanceFixture(t, time.Date(2026, 10, 21, 5, 0, 0, 0, time.UTC))
	require.NoError(t, f.svc.ExpireQuotations(context.Background()))
	assert.Zero(t, f.dashboard.calls)

	f.quotations.n = 3
	require.NoError(t, f.svc.ExpireQuotations(context.Background()))
	assert.Equal(t, 1, f.dashboard.calls)
	assert.Equal(t, time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC), f.quotations.today)
}

func TestPublishScheduledNewsLooksBackOnlyToTheLastRun(t *testing.T) {
	start := time.Date(2026, 10, 21, 5, 0, 0, 0, time.UTC)
	f := newMaintenanceFixture(t, start)

	require.NoError(t, f.svc.PublishScheduledNews(context.Background()))
	assert.Equal(t, start.Add(-time.Minute), f.store.newsFrom, "first run looks one minute back")
	assert.Empty(t, f.site.tags)

	f.now = start.Add(time.Minute)
	f.store.newsCount = 1
	require.NoError(t, f.svc.PublishScheduledNews(context.Background()))
	assert.Equal(t, start, f.store.newsFrom, "later runs continue from the previous run")
	assert.Equal(t, [][]string{{"news", "home", "sitemap"}}, f.site.tags)
}

func TestPurgeFilesRemovesObjectsAndVariantsOfOldDeletedRecords(t *testing.T) {
	f := newMaintenanceFixture(t, time.Date(2026, 10, 21, 5, 0, 0, 0, time.UTC))
	a, b := uuid.New(), uuid.New()
	f.store.batches = [][]model.File{{
		{ID: a, StorageKey: "crew_document/2026/01/a.pdf"},
		{ID: b, StorageKey: "public/news/2026/01/b.webp", Variants: []model.FileVariant{{StorageKey: "public/news/2026/01/b-480.webp"}}},
	}}
	for _, k := range []string{"crew_document/2026/01/a.pdf", "public/news/2026/01/b.webp", "public/news/2026/01/b-480.webp"} {
		f.storage.objects[k] = []byte("x")
	}

	require.NoError(t, f.svc.PurgeFiles(context.Background()))

	assert.Empty(t, f.storage.objects)
	assert.ElementsMatch(t, []uuid.UUID{a, b}, f.store.purgedIDs)
	assert.Equal(t, f.now.AddDate(0, 0, -90), f.store.purgeCutoff)
}

func TestPurgeFilesKeepsRowsUnmarkedWhenObjectDeletionFails(t *testing.T) {
	f := newMaintenanceFixture(t, time.Date(2026, 10, 21, 5, 0, 0, 0, time.UTC))
	f.store.batches = [][]model.File{{{ID: uuid.New(), StorageKey: "a.pdf"}}}
	f.store.purgeErr = nil
	failing := &failingDelete{memBackupStorage: f.storage}
	f.svc.storage = failing

	err := f.svc.PurgeFiles(context.Background())

	require.Error(t, err)
	assert.Empty(t, f.store.purgedIDs)
}

type failingDelete struct{ *memBackupStorage }

func (failingDelete) Delete(context.Context, ...string) error { return errors.New("s3 down") }

func TestRefreshDashboardInvalidatesTheCache(t *testing.T) {
	f := newMaintenanceFixture(t, time.Now())
	require.NoError(t, f.svc.RefreshDashboard(context.Background()))
	assert.Equal(t, 1, f.dashboard.calls)
}

func TestPurgeApplicantsDeletesRecordsAndFilesOlderThanTwentyFourMonths(t *testing.T) {
	f := newMaintenanceFixture(t, time.Date(2026, 10, 21, 5, 0, 0, 0, time.UTC))
	a, cv := uuid.New(), uuid.New()
	f.store.applicants = [][]model.ExpiredApplicant{{{ID: a, Files: []model.File{{ID: cv, StorageKey: "applicant/2024/01/cv.pdf"}}}, {ID: uuid.New()}}}
	f.storage.objects["applicant/2024/01/cv.pdf"] = []byte("x")

	require.NoError(t, f.svc.PurgeApplicants(context.Background()))

	assert.Equal(t, time.Date(2024, 10, 21, 5, 0, 0, 0, time.UTC), f.store.appCutoff)
	assert.Len(t, f.store.deletedApps, 2)
	assert.Equal(t, []uuid.UUID{cv}, f.store.deletedFiles)
	assert.Empty(t, f.storage.objects)
	assert.Equal(t, 1, f.dashboard.calls)
}

func TestPurgeApplicantsKeepsRecordsWhenTheirFilesCannotBeDeleted(t *testing.T) {
	f := newMaintenanceFixture(t, time.Date(2026, 10, 21, 5, 0, 0, 0, time.UTC))
	f.store.applicants = [][]model.ExpiredApplicant{{{ID: uuid.New(), Files: []model.File{{ID: uuid.New(), StorageKey: "a.pdf"}}}}}
	f.svc.storage = &failingDelete{memBackupStorage: f.storage}

	require.Error(t, f.svc.PurgeApplicants(context.Background()))
	assert.Empty(t, f.store.deletedApps)
}
