package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// MaintenanceRepository runs the statements of the scheduled maintenance jobs.
type MaintenanceRepository struct {
	pool *pgxpool.Pool
}

// NewMaintenanceRepository builds a MaintenanceRepository.
func NewMaintenanceRepository(pool *pgxpool.Pool) *MaintenanceRepository {
	return &MaintenanceRepository{pool: pool}
}

// CloseExpiredVacancies closes open vacancies whose closing date is before today.
func (r *MaintenanceRepository) CloseExpiredVacancies(ctx context.Context, today time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE vacancies SET status = 'closed' WHERE status = 'open' AND deleted_at IS NULL AND closing_date < $1`, today)
	if err != nil {
		return 0, fmt.Errorf("close expired vacancies: %w", err)
	}
	return tag.RowsAffected(), nil
}

// CountNewlyLiveNews counts published articles whose publish time fell in (from, to].
func (r *MaintenanceRepository) CountNewlyLiveNews(ctx context.Context, from, to time.Time) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM news WHERE status = 'published' AND deleted_at IS NULL AND published_at > $1 AND published_at <= $2`, from, to).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count newly live news: %w", err)
	}
	return n, nil
}

const purgeCandidatesSQL = `WITH refs(file_id, owner_deleted_at) AS (
	SELECT photo_file_id, deleted_at FROM crews
	UNION ALL SELECT cv_file_id, deleted_at FROM crews
	UNION ALL SELECT td.file_id, c.deleted_at FROM travel_documents td JOIN crews c ON c.id = td.crew_id
	UNION ALL SELECT ce.file_id, c.deleted_at FROM certificates ce JOIN crews c ON c.id = ce.crew_id
	UNION ALL SELECT af.file_id, a.deleted_at FROM applicant_files af JOIN applicants a ON a.id = af.applicant_id
	UNION ALL SELECT gd.file_id, k.deleted_at FROM generated_documents gd JOIN contracts k ON gd.related_type = 'contract' AND k.id = gd.related_id
	UNION ALL SELECT gd.file_id, q.deleted_at FROM generated_documents gd JOIN quotations q ON gd.related_type = 'quotation' AND q.id = gd.related_id
	UNION ALL SELECT gd.file_id, c.deleted_at FROM generated_documents gd JOIN crews c ON gd.related_type = 'crew' AND c.id = gd.related_id
	UNION ALL SELECT thumbnail_file_id, deleted_at FROM news
	UNION ALL SELECT og_image_file_id, deleted_at FROM pages
	UNION ALL SELECT image_file_id, deleted_at FROM hero_slides
	UNION ALL SELECT mobile_image_file_id, deleted_at FROM hero_slides
	UNION ALL SELECT image_file_id, deleted_at FROM gallery_images
	UNION ALL SELECT photo_file_id, deleted_at FROM testimonials
	UNION ALL SELECT logo_file_id, deleted_at FROM client_logos
	UNION ALL SELECT logo_file_id, deleted_at FROM companies
	UNION ALL SELECT flag_file_id, deleted_at FROM languages
)
SELECT f.id, f.storage_key, f.variants FROM files f
WHERE f.purged_at IS NULL
	AND EXISTS (SELECT 1 FROM refs r WHERE r.file_id = f.id AND r.owner_deleted_at < $1)
	AND NOT EXISTS (SELECT 1 FROM refs r WHERE r.file_id = f.id AND (r.owner_deleted_at IS NULL OR r.owner_deleted_at >= $1))
	AND NOT EXISTS (SELECT 1 FROM pages p WHERE p.deleted_at IS NULL AND p.blocks::text LIKE '%' || f.id::text || '%')
	AND NOT EXISTS (SELECT 1 FROM email_logs e WHERE f.id = ANY (e.attachment_file_ids) AND e.created_at >= $1)
ORDER BY f.created_at
LIMIT $2`

// PurgeCandidates returns files used only by records deleted before cutoff.
func (r *MaintenanceRepository) PurgeCandidates(ctx context.Context, cutoff time.Time, limit int) ([]model.File, error) {
	rows, err := r.pool.Query(ctx, purgeCandidatesSQL, cutoff, limit)
	if err != nil {
		return nil, fmt.Errorf("list purgeable files: %w", err)
	}
	files, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.File, error) {
		var f model.File
		err := row.Scan(&f.ID, &f.StorageKey, &f.Variants)
		return f, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan purgeable files: %w", err)
	}
	return files, nil
}

// MarkPurged records that the stored objects of files are gone.
func (r *MaintenanceRepository) MarkPurged(ctx context.Context, ids []uuid.UUID, at time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := r.pool.Exec(ctx, `UPDATE files SET purged_at = $2, deleted_at = coalesce(deleted_at, $2) WHERE id = ANY($1)`, ids, at); err != nil {
		return fmt.Errorf("mark files purged: %w", err)
	}
	return nil
}
