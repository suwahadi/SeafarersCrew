package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

func TestCloseExpiredVacanciesOnlyClosesOpenOnesPastTheirDate(t *testing.T) {
	truncate(t, "vacancies")
	ctx := context.Background()
	exec(t, `INSERT INTO vacancies (public_id, title, slug, status, closing_date) VALUES
		('VAC000001', 'Expired', 'expired', 'open', '2026-10-20'),
		('VAC000002', 'Today', 'today', 'open', '2026-10-21'),
		('VAC000003', 'Open ended', 'open-ended', 'open', NULL),
		('VAC000004', 'Draft', 'draft', 'draft', '2026-10-01')`)
	repo := repository.NewMaintenanceRepository(testDB)

	n, err := repo.CloseExpiredVacancies(ctx, d(2026, 10, 21))

	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
	var statuses []string
	rows, err := testDB.Query(ctx, `SELECT status FROM vacancies ORDER BY public_id`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		statuses = append(statuses, s)
	}
	assert.Equal(t, []string{"closed", "open", "open", "draft"}, statuses)
}

func TestCountNewlyLiveNewsUsesAHalfOpenWindow(t *testing.T) {
	truncate(t, "news")
	exec(t, `INSERT INTO news (title, slug, status, published_at) VALUES
		('Before', 'before', 'published', '2026-10-21 09:59:00+00'),
		('At end', 'at-end', 'published', '2026-10-21 10:01:00+00'),
		('Draft', 'draft', 'draft', '2026-10-21 10:00:30+00'),
		('Later', 'later', 'published', '2026-10-21 10:02:00+00')`)
	repo := repository.NewMaintenanceRepository(testDB)
	from := time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 21, 10, 1, 0, 0, time.UTC)

	n, err := repo.CountNewlyLiveNews(context.Background(), from, to)

	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func purgeFile(t *testing.T, key string) uuid.UUID {
	t.Helper()
	return mustID(t, `INSERT INTO files (storage_key, original_name, mime_type, size_original, size_stored, sha256)
		VALUES ($1, 'f.pdf', 'application/pdf', 10, 10, 'x') RETURNING id`, key)
}

func TestPurgeCandidatesAreFilesUsedOnlyByLongDeletedRecords(t *testing.T) {
	truncate(t, "crews", "news", "pages", "files", "hero_slides", "crew_statuses")
	ctx := context.Background()
	statusID := mustID(t, `INSERT INTO crew_statuses (code, name, is_system) VALUES ('available', 'Available', true) RETURNING id`)
	cutoff := time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC)

	oldPhoto := purgeFile(t, "photo/old.webp")
	exec(t, `INSERT INTO crews (public_id, full_name, crew_status_id, photo_file_id, deleted_at) VALUES ('CREW00001', 'Old', $1, $2, '2026-06-01')`, statusID, oldPhoto)

	recentPhoto := purgeFile(t, "photo/recent.webp")
	exec(t, `INSERT INTO crews (public_id, full_name, crew_status_id, photo_file_id, deleted_at) VALUES ('CREW00002', 'Recent', $1, $2, '2026-09-30')`, statusID, recentPhoto)

	livePhoto := purgeFile(t, "photo/live.webp")
	exec(t, `INSERT INTO crews (public_id, full_name, crew_status_id, photo_file_id) VALUES ('CREW00003', 'Live', $1, $2)`, statusID, livePhoto)

	shared := purgeFile(t, "public/news/shared.webp")
	exec(t, `INSERT INTO news (title, slug, thumbnail_file_id, deleted_at) VALUES ('Gone', 'gone', $1, '2026-05-01')`, shared)
	exec(t, `INSERT INTO hero_slides (image_file_id) VALUES ($1)`, shared)

	inBlock := purgeFile(t, "public/news/in-block.webp")
	exec(t, `INSERT INTO news (title, slug, thumbnail_file_id, deleted_at) VALUES ('Gone 2', 'gone-2', $1, '2026-05-01')`, inBlock)
	exec(t, `INSERT INTO pages (title, slug, blocks) VALUES ('Live page', 'live-page', jsonb_build_array(jsonb_build_object('settings', jsonb_build_object('image_file_id', $1::text))))`, inBlock.String())

	alone := purgeFile(t, "public/news/alone.webp")
	exec(t, `INSERT INTO news (title, slug, thumbnail_file_id, deleted_at) VALUES ('Gone 3', 'gone-3', $1, '2026-05-01')`, alone)

	repo := repository.NewMaintenanceRepository(testDB)
	got, err := repo.PurgeCandidates(ctx, cutoff, 100)
	require.NoError(t, err)
	ids := make([]uuid.UUID, 0, len(got))
	for _, f := range got {
		ids = append(ids, f.ID)
	}
	assert.ElementsMatch(t, []uuid.UUID{oldPhoto, alone}, ids)

	require.NoError(t, repo.MarkPurged(ctx, []uuid.UUID{oldPhoto}, cutoff))
	again, err := repo.PurgeCandidates(ctx, cutoff, 100)
	require.NoError(t, err)
	assert.Len(t, again, 1, "purged files are not offered again")
	var deleted bool
	require.NoError(t, testDB.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM files WHERE id = $1`, oldPhoto).Scan(&deleted))
	assert.True(t, deleted, "a purged file stops resolving through the files API")
}

func TestBackupRepositoryTracksRunsAndRetention(t *testing.T) {
	truncate(t, "backups")
	ctx := context.Background()
	repo := repository.NewBackupRepository(testDB)
	base := time.Date(2026, 10, 21, 2, 0, 0, 0, time.UTC)

	old, err := repo.Create(ctx, base.AddDate(0, 0, -40))
	require.NoError(t, err)
	require.NoError(t, repo.Finish(ctx, old.ID, "succeeded", "backups/old.dump", 1000, "", base.AddDate(0, 0, -40)))
	current, err := repo.Create(ctx, base)
	require.NoError(t, err)
	stuck, err := repo.Create(ctx, base.Add(-5*time.Hour))
	require.NoError(t, err)

	failed, err := repo.FailStale(ctx, base.Add(-3*time.Hour), base)
	require.NoError(t, err)
	assert.EqualValues(t, 1, failed)
	got, err := repo.Get(ctx, stuck.ID)
	require.NoError(t, err)
	assert.Equal(t, "failed", got.Status)
	assert.NotEmpty(t, got.Error)

	expired, err := repo.Expired(ctx, base.AddDate(0, 0, -30))
	require.NoError(t, err)
	require.Len(t, expired, 1)
	assert.Equal(t, "backups/old.dump", expired[0].StorageKey)
	assert.EqualValues(t, 1000, expired[0].SizeBytes)

	recent, err := repo.Since(ctx, base.Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, recent, 1)
	assert.Equal(t, current.ID, recent[0].ID)

	page, err := repo.List(ctx, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, page.Total)
	assert.Len(t, page.Items, 2)
	assert.Equal(t, current.ID, page.Items[0].ID, "newest first")

	require.NoError(t, repo.Delete(ctx, []uuid.UUID{old.ID}))
	_, err = repo.Get(ctx, old.ID)
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestExpiredApplicantsAreDeletedWithTheirHistoryEmailLogsAndFiles(t *testing.T) {
	truncate(t, "applicants", "vacancies", "files", "email_logs")
	ctx := context.Background()
	exec(t, `INSERT INTO vacancies (public_id, title, slug, status) VALUES ('VAC000009', 'V', 'v', 'open')`)
	var vacancyID uuid.UUID
	require.NoError(t, testDB.QueryRow(ctx, `SELECT id FROM vacancies LIMIT 1`).Scan(&vacancyID))
	insert := func(public, email string, created time.Time) uuid.UUID {
		return mustID(t, `INSERT INTO applicants (public_id, vacancy_id, full_name, email, phone, consent_at, created_at)
			VALUES ($1, $2, 'A', $3, '+6281', $4, $4) RETURNING id`, public, vacancyID, email, created)
	}
	oldID := insert("APP000001", "old@example.com", time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC))
	keepID := insert("APP000002", "keep@example.com", time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC))
	cv := purgeFile(t, "applicant/2024/09/cv.pdf")
	exec(t, `INSERT INTO applicant_files (applicant_id, file_id, kind) VALUES ($1, $2, 'cv')`, oldID, cv)
	exec(t, `INSERT INTO applicant_stage_logs (applicant_id, to_stage) VALUES ($1, 'new')`, oldID)
	exec(t, `INSERT INTO email_logs (template_code, to_addresses, subject, status, related_type, related_id) VALUES ('applicant_confirmation', ARRAY['old@example.com'], 's', 'sent', 'applicant', $1)`, oldID)
	repo := repository.NewMaintenanceRepository(testDB)
	cutoff := time.Date(2024, 10, 21, 0, 0, 0, 0, time.UTC)

	expired, err := repo.ExpiredApplicants(ctx, cutoff, 10)
	require.NoError(t, err)
	require.Len(t, expired, 1)
	assert.Equal(t, oldID, expired[0].ID)
	require.Len(t, expired[0].Files, 1)
	assert.Equal(t, "applicant/2024/09/cv.pdf", expired[0].Files[0].StorageKey)

	require.NoError(t, repo.DeleteApplicants(ctx, []uuid.UUID{oldID}, []uuid.UUID{cv}, cutoff))

	var applicants, logs, emails int
	require.NoError(t, testDB.QueryRow(ctx, `SELECT count(*) FROM applicants`).Scan(&applicants))
	require.NoError(t, testDB.QueryRow(ctx, `SELECT count(*) FROM applicant_stage_logs`).Scan(&logs))
	require.NoError(t, testDB.QueryRow(ctx, `SELECT count(*) FROM email_logs`).Scan(&emails))
	assert.Equal(t, 1, applicants)
	assert.Zero(t, logs)
	assert.Zero(t, emails)
	var survivor uuid.UUID
	require.NoError(t, testDB.QueryRow(ctx, `SELECT id FROM applicants`).Scan(&survivor))
	assert.Equal(t, keepID, survivor)
	var purged bool
	require.NoError(t, testDB.QueryRow(ctx, `SELECT purged_at IS NOT NULL FROM files WHERE id = $1`, cv).Scan(&purged))
	assert.True(t, purged)
}
