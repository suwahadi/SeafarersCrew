package repository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

var (
	newsColumns = `id, title, slug, excerpt, body_html, thumbnail_file_id, published_at, status, seo_title, meta_description,
		created_at, updated_at, ` + ImageJSON("thumbnail_file_id")
	newsListColumns = `id, title, slug, excerpt, ''::text, thumbnail_file_id, published_at, status, seo_title, meta_description,
		created_at, updated_at, ` + ImageJSON("thumbnail_file_id")
)

func scanNews(row pgx.Row) (model.News, error) {
	var n model.News
	err := row.Scan(&n.ID, &n.Title, &n.Slug, &n.Excerpt, &n.BodyHTML, &n.ThumbnailFileID, &n.PublishedAt, &n.Status, &n.SEOTitle, &n.MetaDescription,
		&n.CreatedAt, &n.UpdatedAt, &n.Thumbnail)
	return n, err
}

var newsSorts = map[string]string{
	"title": "lower(title)", "status": "status", "published_at": "published_at", "created_at": "created_at", "updated_at": "updated_at",
}

// NewsRepository stores news articles.
type NewsRepository struct {
	pool *pgxpool.Pool
}

// NewNewsRepository returns a NewsRepository.
func NewNewsRepository(pool *pgxpool.Pool) *NewsRepository {
	return &NewsRepository{pool: pool}
}

// List returns one page of live articles without their bodies.
func (r *NewsRepository) List(ctx context.Context, q ListQuery) (Page[model.News], error) {
	w := &Where{}
	w.Add("deleted_at IS NULL")
	if q.Q != "" {
		w.Add("(title ILIKE %[1]s OR slug ILIKE %[1]s OR excerpt ILIKE %[1]s)", LikePattern(q.Q))
	}
	if v, ok := q.Filters["status"]; ok {
		w.Add("status = %s", v)
	}
	from := " FROM news" + w.SQL()
	order := Order(q.Sort, q.Desc, newsSorts, "coalesce(published_at, created_at) DESC, id DESC")
	limit := " LIMIT " + strconv.Itoa(q.PerPage) + " OFFSET " + strconv.Itoa((q.Page-1)*q.PerPage)

	var page Page[model.News]
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		rows, err := r.pool.Query(gctx, "SELECT "+newsListColumns+from+order+limit, w.Args()...)
		if err != nil {
			return fmt.Errorf("list news: %w", err)
		}
		items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.News, error) { return scanNews(row) })
		if err != nil {
			return fmt.Errorf("scan news: %w", err)
		}
		page.Items = items
		return nil
	})
	g.Go(func() error {
		if err := r.pool.QueryRow(gctx, "SELECT count(*)"+from, w.Args()...).Scan(&page.Total); err != nil {
			return fmt.Errorf("count news: %w", err)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return Page[model.News]{}, err
	}
	return page, nil
}

// Get returns a live article by ID.
func (r *NewsRepository) Get(ctx context.Context, id uuid.UUID) (model.News, error) {
	n, err := scanNews(r.pool.QueryRow(ctx, `SELECT `+newsColumns+` FROM news WHERE id = $1 AND deleted_at IS NULL`, id))
	if err != nil {
		return n, fmt.Errorf("get news: %w", notFound(err))
	}
	return n, nil
}

// SlugTaken reports whether another live article uses slug.
func (r *NewsRepository) SlugTaken(ctx context.Context, slug string, exclude uuid.UUID) (bool, error) {
	return SlugTaken(ctx, r.pool, "news", slug, exclude)
}

// Insert stores a new article.
func (r *NewsRepository) Insert(ctx context.Context, n model.News) (model.News, error) {
	created, err := scanNews(r.pool.QueryRow(ctx, `INSERT INTO news
		(title, slug, excerpt, body_html, thumbnail_file_id, published_at, status, seo_title, meta_description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING `+newsColumns,
		n.Title, n.Slug, n.Excerpt, n.BodyHTML, n.ThumbnailFileID, n.PublishedAt, n.Status, n.SEOTitle, n.MetaDescription))
	if err != nil {
		return created, fmt.Errorf("insert news: %w", duplicate(err))
	}
	return created, nil
}

// Update overwrites a live article; redirectFrom, when set, is the old path that must 301 to the new one.
func (r *NewsRepository) Update(ctx context.Context, id uuid.UUID, n model.News, redirectFrom string) (model.News, error) {
	var updated model.News
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		updated, err = scanNews(tx.QueryRow(ctx, `UPDATE news SET title = $2, slug = $3, excerpt = $4, body_html = $5, thumbnail_file_id = $6,
			published_at = $7, status = $8, seo_title = $9, meta_description = $10
			WHERE id = $1 AND deleted_at IS NULL RETURNING `+newsColumns,
			id, n.Title, n.Slug, n.Excerpt, n.BodyHTML, n.ThumbnailFileID, n.PublishedAt, n.Status, n.SEOTitle, n.MetaDescription))
		if err != nil {
			return fmt.Errorf("update news: %w", duplicate(notFound(err)))
		}
		if redirectFrom != "" {
			return SaveRedirect(ctx, tx, redirectFrom, "/news/"+updated.Slug)
		}
		return nil
	})
	return updated, err
}

// SoftDelete marks an article deleted.
func (r *NewsRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE news SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("delete news: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
