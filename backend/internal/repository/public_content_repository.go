package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

const liveNews = `status = 'published' AND deleted_at IS NULL AND published_at IS NOT NULL AND published_at <= $1`

const openVacancy = `v.status = 'open' AND v.deleted_at IS NULL AND (v.closing_date IS NULL OR v.closing_date >= $1)`

// PublicContentRepository reads published content for the public API.
type PublicContentRepository struct {
	pool *pgxpool.Pool
}

// NewPublicContentRepository returns a PublicContentRepository.
func NewPublicContentRepository(pool *pgxpool.Pool) *PublicContentRepository {
	return &PublicContentRepository{pool: pool}
}

// PublishedPage returns the published page with slug, system pages included.
func (r *PublicContentRepository) PublishedPage(ctx context.Context, slug string) (model.Page, error) {
	p, err := scanPage(r.pool.QueryRow(ctx, `SELECT `+pageColumns+` FROM pages WHERE slug = $1 AND status = 'published' AND deleted_at IS NULL`, slug))
	if err != nil {
		return p, fmt.Errorf("get published page: %w", notFound(err))
	}
	return p, nil
}

// SystemPage returns a system page by key.
func (r *PublicContentRepository) SystemPage(ctx context.Context, key string) (model.Page, error) {
	p, err := scanPage(r.pool.QueryRow(ctx, `SELECT `+pageColumns+` FROM pages WHERE system_key = $1 AND deleted_at IS NULL`, key))
	if err != nil {
		return p, fmt.Errorf("get system page: %w", notFound(err))
	}
	return p, nil
}

// GalleryImagesByID returns the active gallery images among ids in the order of ids.
func (r *PublicContentRepository) GalleryImagesByID(ctx context.Context, ids []uuid.UUID) ([]model.GalleryImage, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT `+galleryColumns+` FROM gallery_images WHERE id = ANY($1) AND is_active AND deleted_at IS NULL`, ids)
	if err != nil {
		return nil, fmt.Errorf("list gallery images by id: %w", err)
	}
	found, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.GalleryImage, error) { return scanGalleryImage(row) })
	if err != nil {
		return nil, fmt.Errorf("scan gallery images: %w", err)
	}
	byID := make(map[uuid.UUID]model.GalleryImage, len(found))
	for _, g := range found {
		byID[g.ID] = g
	}
	out := make([]model.GalleryImage, 0, len(found))
	for _, id := range ids {
		if g, ok := byID[id]; ok {
			out = append(out, g)
		}
	}
	return out, nil
}

// LiveNews returns one page of articles visible at now, newest first, without bodies.
func (r *PublicContentRepository) LiveNews(ctx context.Context, now time.Time, page, perPage int) (Page[model.News], error) {
	var out Page[model.News]
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		rows, err := r.pool.Query(gctx, `SELECT `+newsListColumns+` FROM news WHERE `+liveNews+`
			ORDER BY published_at DESC, id DESC LIMIT `+strconv.Itoa(perPage)+` OFFSET `+strconv.Itoa((page-1)*perPage), now)
		if err != nil {
			return fmt.Errorf("list live news: %w", err)
		}
		items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.News, error) { return scanNews(row) })
		if err != nil {
			return fmt.Errorf("scan live news: %w", err)
		}
		out.Items = items
		return nil
	})
	g.Go(func() error {
		if err := r.pool.QueryRow(gctx, `SELECT count(*) FROM news WHERE `+liveNews, now).Scan(&out.Total); err != nil {
			return fmt.Errorf("count live news: %w", err)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return Page[model.News]{}, err
	}
	return out, nil
}

// LatestNews returns up to limit visible articles other than exclude, newest first, without bodies.
func (r *PublicContentRepository) LatestNews(ctx context.Context, now time.Time, limit int, exclude uuid.UUID) ([]model.News, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+newsListColumns+` FROM news WHERE `+liveNews+` AND id <> $2
		ORDER BY published_at DESC, id DESC LIMIT `+strconv.Itoa(limit), now, exclude)
	if err != nil {
		return nil, fmt.Errorf("list latest news: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.News, error) { return scanNews(row) })
	if err != nil {
		return nil, fmt.Errorf("scan latest news: %w", err)
	}
	return items, nil
}

// LiveNewsBySlug returns the article with slug when it is visible at now.
func (r *PublicContentRepository) LiveNewsBySlug(ctx context.Context, now time.Time, slug string) (model.News, error) {
	n, err := scanNews(r.pool.QueryRow(ctx, `SELECT `+newsColumns+` FROM news WHERE `+liveNews+` AND slug = $2`, now, slug))
	if err != nil {
		return n, fmt.Errorf("get live news: %w", notFound(err))
	}
	return n, nil
}

// OpenVacancies returns up to limit open vacancies not past their closing date, newest first.
func (r *PublicContentRepository) OpenVacancies(ctx context.Context, today time.Time, limit int) ([]model.VacancyCard, error) {
	rows, err := r.pool.Query(ctx, `SELECT v.public_id, v.title, v.slug, coalesce(rk.name, ''), coalesce(vt.name, ''), v.location, v.closing_date
		FROM vacancies v
		LEFT JOIN ranks rk ON rk.id = v.rank_id
		LEFT JOIN vessel_types vt ON vt.id = v.vessel_type_id
		WHERE `+openVacancy+`
		ORDER BY coalesce(v.published_at, v.created_at) DESC, v.id DESC LIMIT `+strconv.Itoa(limit), today)
	if err != nil {
		return nil, fmt.Errorf("list open vacancies: %w", err)
	}
	cards, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.VacancyCard, error) {
		var c model.VacancyCard
		err := row.Scan(&c.PublicID, &c.Title, &c.Slug, &c.RankName, &c.VesselTypeName, &c.Location, &c.ClosingDate)
		return c, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan open vacancies: %w", err)
	}
	return cards, nil
}

// SitemapPages returns the published pages and the system pages with their last modification times.
func (r *PublicContentRepository) SitemapPages(ctx context.Context) (pages, system []model.SitemapEntry, err error) {
	rows, err := r.pool.Query(ctx, `SELECT slug, system_key, updated_at FROM pages WHERE status = 'published' AND deleted_at IS NULL ORDER BY is_system DESC, menu_order, slug`)
	if err != nil {
		return nil, nil, fmt.Errorf("list sitemap pages: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			slug      string
			systemKey *string
			updated   time.Time
		)
		if err := rows.Scan(&slug, &systemKey, &updated); err != nil {
			return nil, nil, fmt.Errorf("scan sitemap pages: %w", err)
		}
		if systemKey != nil {
			system = append(system, model.SitemapEntry{Path: model.SystemPath(*systemKey), LastMod: updated})
			continue
		}
		pages = append(pages, model.SitemapEntry{Path: "/" + slug, LastMod: updated})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("read sitemap pages: %w", err)
	}
	return pages, system, nil
}

// SitemapNews returns the visible articles with their last modification times.
func (r *PublicContentRepository) SitemapNews(ctx context.Context, now time.Time) ([]model.SitemapEntry, error) {
	return r.sitemapEntries(ctx, `SELECT '/news/' || slug, updated_at FROM news WHERE `+liveNews+` ORDER BY published_at DESC, id DESC`, now)
}

// SitemapVacancies returns the open vacancies with their last modification times.
func (r *PublicContentRepository) SitemapVacancies(ctx context.Context, today time.Time) ([]model.SitemapEntry, error) {
	return r.sitemapEntries(ctx, `SELECT '/career/' || v.slug, v.updated_at FROM vacancies v WHERE `+openVacancy+` ORDER BY coalesce(v.published_at, v.created_at) DESC, v.id DESC`, today)
}

func (r *PublicContentRepository) sitemapEntries(ctx context.Context, sql string, arg any) ([]model.SitemapEntry, error) {
	rows, err := r.pool.Query(ctx, sql, arg)
	if err != nil {
		return nil, fmt.Errorf("list sitemap entries: %w", err)
	}
	entries, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.SitemapEntry, error) {
		var e model.SitemapEntry
		err := row.Scan(&e.Path, &e.LastMod)
		return e, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan sitemap entries: %w", err)
	}
	return entries, nil
}
