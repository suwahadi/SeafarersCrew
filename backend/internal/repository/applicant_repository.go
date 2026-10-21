package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// ErrDuplicateApplication is returned when the same email already applied to a vacancy within the guard window.
var ErrDuplicateApplication = errors.New("duplicate application")

// ApplicantRepository stores applicants, their files and their stage history.
type ApplicantRepository struct {
	pool *pgxpool.Pool
}

// NewApplicantRepository returns an ApplicantRepository.
func NewApplicantRepository(pool *pgxpool.Pool) *ApplicantRepository {
	return &ApplicantRepository{pool: pool}
}

const applicantFrom = `
FROM applicants a
JOIN vacancies v ON v.id = a.vacancy_id
LEFT JOIN ranks r ON r.id = a.rank_applied_id`

const applicantColumns = `a.id, a.public_id, a.vacancy_id, v.public_id, v.title, v.slug, v.status, a.full_name, a.email, a.phone,
	a.date_of_birth, a.nationality, a.rank_applied_id, coalesce(r.name, ''), a.years_experience, a.message, a.stage, a.consent_at,
	coalesce(host(a.ip_address), ''), (SELECT count(*) FROM applicant_files af WHERE af.applicant_id = a.id), a.created_at, a.updated_at`

func scanApplicant(row pgx.Row) (model.Applicant, error) {
	var a model.Applicant
	var years *int16
	err := row.Scan(&a.ID, &a.PublicID, &a.VacancyID, &a.VacancyPublicID, &a.VacancyTitle, &a.VacancySlug, &a.VacancyStatus,
		&a.FullName, &a.Email, &a.Phone, &a.DateOfBirth, &a.Nationality, &a.RankAppliedID, &a.RankAppliedName, &years,
		&a.Message, &a.Stage, &a.ConsentAt, &a.IPAddress, &a.FilesCount, &a.CreatedAt, &a.UpdatedAt)
	a.YearsExperience = intPtr(years)
	return a, notFound(err)
}

// Create stores an application with its files and first stage log, refusing a duplicate inside the guard window.
func (r *ApplicantRepository) Create(ctx context.Context, n model.NewApplication) error {
	return InTx(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, n.VacancyID.String()+"|"+strings.ToLower(n.Email)); err != nil {
			return fmt.Errorf("lock application guard: %w", err)
		}
		duplicate, err := recentApplication(ctx, tx, n)
		if err != nil {
			return err
		}
		if duplicate {
			return ErrDuplicateApplication
		}
		if _, err := tx.Exec(ctx, `INSERT INTO applicants (id, public_id, vacancy_id, full_name, email, phone, date_of_birth, nationality,
			rank_applied_id, years_experience, message, stage, consent_at, ip_address)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'new', $12, nullif($13, '')::inet)`,
			n.ID, n.PublicID, n.VacancyID, n.FullName, n.Email, n.Phone, n.DateOfBirth, n.Nationality, n.RankAppliedID,
			n.YearsExperience, n.Message, n.ConsentAt, n.IPAddress); err != nil {
			return fmt.Errorf("insert applicant: %w", err)
		}
		for _, f := range n.Files {
			if _, err := tx.Exec(ctx, `INSERT INTO applicant_files (applicant_id, file_id, kind) VALUES ($1, $2, $3)`, n.ID, f.FileID, f.Kind); err != nil {
				return fmt.Errorf("insert applicant file: %w", err)
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO applicant_stage_logs (applicant_id, from_stage, to_stage, note) VALUES ($1, NULL, 'new', '')`, n.ID); err != nil {
			return fmt.Errorf("insert first stage log: %w", err)
		}
		return nil
	})
}

func recentApplication(ctx context.Context, db DBTX, n model.NewApplication) (bool, error) {
	var found bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM applicants WHERE vacancy_id = $1 AND lower(email) = lower($2)
		AND created_at >= $3 AND deleted_at IS NULL)`, n.VacancyID, n.Email, n.DuplicateSince).Scan(&found)
	if err != nil {
		return false, fmt.Errorf("check duplicate application: %w", err)
	}
	return found, nil
}

// HasRecent reports whether the email already applied to the vacancy since the guard window started.
func (r *ApplicantRepository) HasRecent(ctx context.Context, n model.NewApplication) (bool, error) {
	return recentApplication(ctx, r.pool, n)
}

// GetByPublicID returns an applicant with its files and stage history.
func (r *ApplicantRepository) GetByPublicID(ctx context.Context, publicID string) (model.Applicant, error) {
	a, err := scanApplicant(r.pool.QueryRow(ctx, `SELECT `+applicantColumns+applicantFrom+` WHERE a.public_id = $1 AND a.deleted_at IS NULL`, publicID))
	if err != nil {
		return a, fmt.Errorf("get applicant: %w", err)
	}
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		a.Files, err = r.files(gctx, a.ID)
		return err
	})
	g.Go(func() error {
		var err error
		a.StageLogs, err = r.stageLogs(gctx, a.ID)
		return err
	})
	return a, g.Wait()
}

func (r *ApplicantRepository) files(ctx context.Context, applicantID uuid.UUID) ([]model.ApplicantFile, error) {
	rows, err := r.pool.Query(ctx, `SELECT f.id, af.kind, f.original_name, f.storage_key, f.mime_type, f.size_stored, af.created_at
		FROM applicant_files af JOIN files f ON f.id = af.file_id
		WHERE af.applicant_id = $1 AND f.deleted_at IS NULL ORDER BY (af.kind = 'cv') DESC, af.created_at, af.id`, applicantID)
	if err != nil {
		return nil, fmt.Errorf("list applicant files: %w", err)
	}
	return collect(rows, func(row pgx.Row) (model.ApplicantFile, error) {
		var f model.ApplicantFile
		err := row.Scan(&f.FileID, &f.Kind, &f.OriginalName, &f.StorageKey, &f.MimeType, &f.Size, &f.CreatedAt)
		return f, err
	}, "applicant files")
}

func (r *ApplicantRepository) stageLogs(ctx context.Context, applicantID uuid.UUID) ([]model.StageLog, error) {
	rows, err := r.pool.Query(ctx, `SELECT l.id, coalesce(l.from_stage, ''), l.to_stage, l.note, l.admin_id, coalesce(ad.name, ''), l.created_at
		FROM applicant_stage_logs l LEFT JOIN admins ad ON ad.id = l.admin_id
		WHERE l.applicant_id = $1 ORDER BY l.created_at, l.id`, applicantID)
	if err != nil {
		return nil, fmt.Errorf("list stage logs: %w", err)
	}
	return collect(rows, func(row pgx.Row) (model.StageLog, error) {
		var l model.StageLog
		err := row.Scan(&l.ID, &l.FromStage, &l.ToStage, &l.Note, &l.AdminID, &l.AdminName, &l.CreatedAt)
		return l, err
	}, "stage logs")
}

func applicantWhere(f model.ApplicantFilter, withStages bool) *Where {
	w := &Where{}
	w.Add(`a.deleted_at IS NULL`)
	if f.Q != "" {
		w.Add(`(a.full_name ILIKE %[1]s OR a.email ILIKE %[1]s OR a.public_id ILIKE %[1]s)`, LikePattern(f.Q))
	}
	if f.VacancyPublic != "" {
		w.Add(`v.public_id = %s`, f.VacancyPublic)
	}
	if withStages && len(f.Stages) > 0 {
		w.Add(`a.stage = ANY(%s)`, f.Stages)
	}
	if len(f.RankIDs) > 0 {
		w.Add(`a.rank_applied_id = ANY(%s)`, f.RankIDs)
	}
	if f.CreatedFrom != nil {
		w.Add(`a.created_at >= %s`, *f.CreatedFrom)
	}
	if f.CreatedBefore != nil {
		w.Add(`a.created_at < %s`, *f.CreatedBefore)
	}
	return w
}

// List returns one page of applicants matching f, newest first unless sorted.
func (r *ApplicantRepository) List(ctx context.Context, f model.ApplicantFilter, limit, offset int) (Page[model.Applicant], error) {
	w := applicantWhere(f, true)
	var page Page[model.Applicant]
	if err := r.pool.QueryRow(ctx, `SELECT count(*)`+applicantFrom+w.SQL(), w.Args()...).Scan(&page.Total); err != nil {
		return page, fmt.Errorf("count applicants: %w", err)
	}
	cols := map[string]string{
		"id": "a.public_id", "full_name": "a.full_name", "stage": "a.stage", "created_at": "a.created_at",
		"vacancy": "v.title", "rank": "r.name", "years_experience": "a.years_experience",
	}
	lim, off := w.Arg(limit), w.Arg(offset)
	rows, err := r.pool.Query(ctx, `SELECT `+applicantColumns+applicantFrom+w.SQL()+
		OrderWithTie(f.Sort, f.Desc, cols, "a.created_at DESC, a.id DESC", "a.id")+` LIMIT `+lim+` OFFSET `+off, w.Args()...)
	if err != nil {
		return page, fmt.Errorf("list applicants: %w", err)
	}
	page.Items, err = collect(rows, scanApplicant, "applicants")
	return page, err
}

// StageCounts returns the number of applicants per stage for f, ignoring its stage filter.
func (r *ApplicantRepository) StageCounts(ctx context.Context, f model.ApplicantFilter) (map[string]int, error) {
	w := applicantWhere(f, false)
	rows, err := r.pool.Query(ctx, `SELECT a.stage, count(*)`+applicantFrom+w.SQL()+` GROUP BY a.stage`, w.Args()...)
	if err != nil {
		return nil, fmt.Errorf("count applicants per stage: %w", err)
	}
	type stageCount struct {
		stage string
		n     int
	}
	pairs, err := collect(rows, func(row pgx.Row) (stageCount, error) {
		var c stageCount
		err := row.Scan(&c.stage, &c.n)
		return c, err
	}, "stage counts")
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int, len(pairs))
	for _, p := range pairs {
		counts[p.stage] = p.n
	}
	return counts, nil
}

// ChangeStage moves an applicant out of from and records the move; it returns ErrStateChanged when the stage is no longer from.
func (r *ApplicantRepository) ChangeStage(ctx context.Context, id uuid.UUID, from, to, note string, adminID uuid.UUID) error {
	return InTx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE applicants SET stage = $3 WHERE id = $1 AND stage = $2 AND deleted_at IS NULL`, id, from, to)
		if err != nil {
			return fmt.Errorf("update applicant stage: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrStateChanged
		}
		if _, err := tx.Exec(ctx, `INSERT INTO applicant_stage_logs (applicant_id, from_stage, to_stage, note, admin_id) VALUES ($1, $2, $3, $4, $5)`,
			id, from, to, note, adminID); err != nil {
			return fmt.Errorf("insert stage log: %w", err)
		}
		return nil
	})
}

// SoftDelete marks an applicant deleted.
func (r *ApplicantRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE applicants SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return affectedOne(tag, err, "delete applicant")
}
