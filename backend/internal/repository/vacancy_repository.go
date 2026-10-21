package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// VacancyRepository stores job vacancies.
type VacancyRepository struct {
	pool *pgxpool.Pool
}

// NewVacancyRepository returns a VacancyRepository.
func NewVacancyRepository(pool *pgxpool.Pool) *VacancyRepository {
	return &VacancyRepository{pool: pool}
}

const vacancyFrom = `
FROM vacancies v
JOIN ranks r ON r.id = v.rank_id
LEFT JOIN vessel_types vt ON vt.id = v.vessel_type_id`

const vacancyIdentity = `v.id, v.public_id, v.title, v.slug, v.rank_id, r.name, v.vessel_type_id, coalesce(vt.name, ''),
	coalesce(v.department, ''), v.location, v.employment_type, v.duration, v.salary_text,`

const vacancyTimes = `v.closing_date, v.status, v.published_at,`

const vacancyAdminColumns = vacancyIdentity + ` v.description_html, v.requirements_html, v.required_documents, ` + vacancyTimes + `
	(SELECT count(*) FROM applicants a WHERE a.vacancy_id = v.id AND a.deleted_at IS NULL),
	(SELECT count(*) FROM applicants a WHERE a.vacancy_id = v.id AND a.deleted_at IS NULL AND a.stage = 'new'),
	v.created_at, v.updated_at`

const vacancyDetailColumns = vacancyIdentity + ` v.description_html, v.requirements_html, v.required_documents, ` + vacancyTimes + `
	0, 0, v.created_at, v.updated_at`

const vacancyCardColumns = vacancyIdentity + ` '', '', '[]'::jsonb, ` + vacancyTimes + `
	0, 0, v.created_at, v.updated_at`

func scanVacancy(row pgx.Row) (model.Vacancy, error) {
	var v model.Vacancy
	err := row.Scan(&v.ID, &v.PublicID, &v.Title, &v.Slug, &v.RankID, &v.RankName, &v.VesselTypeID, &v.VesselTypeName,
		&v.Department, &v.Location, &v.EmploymentType, &v.Duration, &v.SalaryText, &v.DescriptionHTML, &v.RequirementsHTML,
		&v.RequiredDocuments, &v.ClosingDate, &v.Status, &v.PublishedAt, &v.ApplicationsCount, &v.NewApplicationsCount,
		&v.CreatedAt, &v.UpdatedAt)
	if v.RequiredDocuments == nil {
		v.RequiredDocuments = []string{}
	}
	return v, notFound(err)
}

// Create inserts a vacancy.
func (r *VacancyRepository) Create(ctx context.Context, v model.Vacancy, publicID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `INSERT INTO vacancies (public_id, title, slug, rank_id, vessel_type_id, department, location,
		employment_type, duration, salary_text, description_html, requirements_html, required_documents, closing_date, status, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) RETURNING id`,
		publicID, v.Title, v.Slug, v.RankID, v.VesselTypeID, v.Department, v.Location, v.EmploymentType, v.Duration,
		v.SalaryText, v.DescriptionHTML, v.RequirementsHTML, v.RequiredDocuments, v.ClosingDate, v.Status, v.PublishedAt).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert vacancy: %w", err)
	}
	return id, nil
}

// Update replaces the editable fields of a vacancy and stores redirect in the same transaction when given.
func (r *VacancyRepository) Update(ctx context.Context, v model.Vacancy, redirect *model.Redirect) error {
	return InTx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE vacancies SET title = $2, slug = $3, rank_id = $4, vessel_type_id = $5, department = $6,
			location = $7, employment_type = $8, duration = $9, salary_text = $10, description_html = $11, requirements_html = $12,
			required_documents = $13, closing_date = $14, status = $15, published_at = $16
			WHERE id = $1 AND deleted_at IS NULL`,
			v.ID, v.Title, v.Slug, v.RankID, v.VesselTypeID, v.Department, v.Location, v.EmploymentType, v.Duration,
			v.SalaryText, v.DescriptionHTML, v.RequirementsHTML, v.RequiredDocuments, v.ClosingDate, v.Status, v.PublishedAt)
		if err := affectedOne(tag, err, "update vacancy"); err != nil {
			return err
		}
		if redirect == nil {
			return nil
		}
		return SaveRedirect(ctx, tx, redirect.FromPath, redirect.ToPath)
	})
}

// GetByPublicID returns a non-deleted vacancy with its application counts.
func (r *VacancyRepository) GetByPublicID(ctx context.Context, publicID string) (model.Vacancy, error) {
	return r.get(ctx, vacancyAdminColumns, `v.public_id = $1`, publicID)
}

// GetByID returns a non-deleted vacancy with its application counts.
func (r *VacancyRepository) GetByID(ctx context.Context, id uuid.UUID) (model.Vacancy, error) {
	return r.get(ctx, vacancyAdminColumns, `v.id = $1`, id)
}

// GetBySlug returns a non-deleted vacancy of any status for the public detail page.
func (r *VacancyRepository) GetBySlug(ctx context.Context, slug string) (model.Vacancy, error) {
	return r.get(ctx, vacancyDetailColumns, `v.slug = $1`, slug)
}

func (r *VacancyRepository) get(ctx context.Context, columns, cond string, arg any) (model.Vacancy, error) {
	v, err := scanVacancy(r.pool.QueryRow(ctx, `SELECT `+columns+vacancyFrom+` WHERE `+cond+` AND v.deleted_at IS NULL`, arg))
	if err != nil {
		return v, fmt.Errorf("get vacancy: %w", err)
	}
	return v, nil
}

// SoftDelete marks a vacancy deleted; its applicants keep the reference.
func (r *VacancyRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE vacancies SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return affectedOne(tag, err, "delete vacancy")
}

// SlugTaken reports whether another live vacancy uses slug.
func (r *VacancyRepository) SlugTaken(ctx context.Context, slug string, exclude uuid.UUID) (bool, error) {
	return SlugTaken(ctx, r.pool, "vacancies", slug, exclude)
}

// RankActive reports whether an active, non-deleted rank exists.
func (r *VacancyRepository) RankActive(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.exists(ctx, `SELECT EXISTS (SELECT 1 FROM ranks WHERE id = $1 AND deleted_at IS NULL AND is_active)`, id)
}

// VesselTypeActive reports whether an active, non-deleted vessel type exists.
func (r *VacancyRepository) VesselTypeActive(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.exists(ctx, `SELECT EXISTS (SELECT 1 FROM vessel_types WHERE id = $1 AND deleted_at IS NULL AND is_active)`, id)
}

func (r *VacancyRepository) exists(ctx context.Context, sql string, id uuid.UUID) (bool, error) {
	var ok bool
	if err := r.pool.QueryRow(ctx, sql, id).Scan(&ok); err != nil {
		return false, fmt.Errorf("vacancy lookup: %w", err)
	}
	return ok, nil
}

func vacancyWhere(f model.VacancyFilter) *Where {
	w := &Where{}
	w.Add(`v.deleted_at IS NULL`)
	if f.Q != "" {
		p := LikePattern(f.Q)
		w.Add(`(v.title ILIKE %[1]s OR v.public_id ILIKE %[1]s)`, p)
	}
	if len(f.Statuses) > 0 {
		w.Add(`v.status = ANY(%s)`, f.Statuses)
	}
	if len(f.RankIDs) > 0 {
		w.Add(`v.rank_id = ANY(%s)`, f.RankIDs)
	}
	if len(f.VesselTypeIDs) > 0 {
		w.Add(`v.vessel_type_id = ANY(%s)`, f.VesselTypeIDs)
	}
	if len(f.Departments) > 0 {
		w.Add(`v.department = ANY(%s)`, f.Departments)
	}
	return w
}

// List returns one page of vacancies matching f.
func (r *VacancyRepository) List(ctx context.Context, f model.VacancyFilter, limit, offset int) (Page[model.Vacancy], error) {
	w := vacancyWhere(f)
	var page Page[model.Vacancy]
	if err := r.pool.QueryRow(ctx, `SELECT count(*)`+vacancyFrom+w.SQL(), w.Args()...).Scan(&page.Total); err != nil {
		return page, fmt.Errorf("count vacancies: %w", err)
	}
	cols := map[string]string{
		"id": "v.public_id", "title": "v.title", "status": "v.status", "department": "v.department", "rank": "r.name",
		"closing_date": "v.closing_date", "published_at": "v.published_at", "created_at": "v.created_at",
		"applications_count": "(SELECT count(*) FROM applicants a WHERE a.vacancy_id = v.id AND a.deleted_at IS NULL)",
	}
	lim, off := w.Arg(limit), w.Arg(offset)
	rows, err := r.pool.Query(ctx, `SELECT `+vacancyAdminColumns+vacancyFrom+w.SQL()+
		OrderWithTie(f.Sort, f.Desc, cols, "v.created_at DESC, v.id DESC", "v.id")+` LIMIT `+lim+` OFFSET `+off, w.Args()...)
	if err != nil {
		return page, fmt.Errorf("list vacancies: %w", err)
	}
	page.Items, err = collect(rows, scanVacancy, "vacancies")
	return page, err
}

func openVacancyWhere(f model.PublicVacancyFilter) *Where {
	w := &Where{}
	w.Add(`v.deleted_at IS NULL AND v.status = 'open' AND (v.closing_date IS NULL OR v.closing_date >= %s)`, f.Today)
	if f.Q != "" {
		p := LikePattern(f.Q)
		w.Add(`(v.title ILIKE %[1]s OR r.name ILIKE %[1]s OR v.description_html ILIKE %[1]s)`, p)
	}
	if f.RankID != nil {
		w.Add(`v.rank_id = %s`, *f.RankID)
	}
	if f.VesselTypeID != nil {
		w.Add(`v.vessel_type_id = %s`, *f.VesselTypeID)
	}
	if f.Department != "" {
		w.Add(`v.department = %s`, f.Department)
	}
	if f.Location != "" {
		w.Add(`lower(v.location) = lower(%s)`, f.Location)
	}
	return w
}

// ListOpen returns one page of vacancies that are open for applications on f.Today, newest first.
func (r *VacancyRepository) ListOpen(ctx context.Context, f model.PublicVacancyFilter, limit, offset int) (Page[model.Vacancy], error) {
	w := openVacancyWhere(f)
	var page Page[model.Vacancy]
	if err := r.pool.QueryRow(ctx, `SELECT count(*)`+vacancyFrom+w.SQL(), w.Args()...).Scan(&page.Total); err != nil {
		return page, fmt.Errorf("count open vacancies: %w", err)
	}
	lim, off := w.Arg(limit), w.Arg(offset)
	rows, err := r.pool.Query(ctx, `SELECT `+vacancyCardColumns+vacancyFrom+w.SQL()+
		` ORDER BY v.published_at DESC NULLS LAST, v.id DESC LIMIT `+lim+` OFFSET `+off, w.Args()...)
	if err != nil {
		return page, fmt.Errorf("list open vacancies: %w", err)
	}
	page.Items, err = collect(rows, scanVacancy, "open vacancies")
	return page, err
}

// FilterOptions returns the ranks, departments, vessel types and locations used by vacancies open on today.
func (r *VacancyRepository) FilterOptions(ctx context.Context, today time.Time) (model.VacancyFilterOptions, error) {
	const open = `v.deleted_at IS NULL AND v.status = 'open' AND (v.closing_date IS NULL OR v.closing_date >= $1)`
	var out model.VacancyFilterOptions
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		rows, err := r.pool.Query(gctx, `SELECT DISTINCT r.id, r.name, r.short_code, r.sort_order FROM vacancies v
			JOIN ranks r ON r.id = v.rank_id WHERE `+open+` ORDER BY r.sort_order, r.name`, today)
		if err != nil {
			return fmt.Errorf("list vacancy ranks: %w", err)
		}
		out.Ranks, err = collect(rows, scanOption, "vacancy ranks")
		return err
	})
	g.Go(func() error {
		rows, err := r.pool.Query(gctx, `SELECT DISTINCT vt.id, vt.name, '', vt.sort_order FROM vacancies v
			JOIN vessel_types vt ON vt.id = v.vessel_type_id WHERE `+open+` ORDER BY vt.sort_order, vt.name`, today)
		if err != nil {
			return fmt.Errorf("list vacancy vessel types: %w", err)
		}
		out.VesselTypes, err = collect(rows, scanOption, "vacancy vessel types")
		return err
	})
	g.Go(func() error {
		rows, err := r.pool.Query(gctx, `SELECT DISTINCT v.department FROM vacancies v
			WHERE `+open+` AND v.department IS NOT NULL ORDER BY v.department`, today)
		if err != nil {
			return fmt.Errorf("list vacancy departments: %w", err)
		}
		out.Departments, err = collect(rows, scanString, "vacancy departments")
		return err
	})
	g.Go(func() error {
		rows, err := r.pool.Query(gctx, `SELECT DISTINCT v.location FROM vacancies v
			WHERE `+open+` AND v.location <> '' ORDER BY v.location`, today)
		if err != nil {
			return fmt.Errorf("list vacancy locations: %w", err)
		}
		out.Locations, err = collect(rows, scanString, "vacancy locations")
		return err
	})
	if err := g.Wait(); err != nil {
		return model.VacancyFilterOptions{}, err
	}
	return out, nil
}

func scanOption(row pgx.Row) (model.Option, error) {
	var o model.Option
	var sort int
	err := row.Scan(&o.ID, &o.Name, &o.ShortCode, &sort)
	return o, err
}

func scanString(row pgx.Row) (string, error) {
	var s string
	err := row.Scan(&s)
	return s, err
}
