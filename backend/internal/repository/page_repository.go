package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

const pageFields = `id, title, slug, system_key, intro_html, %s, seo_title, meta_description, og_image_file_id, status,
	in_header_menu, in_footer_menu, menu_order, is_system, published_at, created_at, updated_at`

var (
	pageColumns     = fmt.Sprintf(pageFields, "blocks") + ", " + ImageJSON("og_image_file_id")
	pageListColumns = fmt.Sprintf(pageFields, "'[]'::jsonb") + ", " + ImageJSON("og_image_file_id")
)

func scanPage(row pgx.Row) (model.Page, error) {
	var p model.Page
	err := row.Scan(&p.ID, &p.Title, &p.Slug, &p.SystemKey, &p.IntroHTML, &p.Blocks, &p.SEOTitle, &p.MetaDescription, &p.OGImageFileID, &p.Status,
		&p.InHeaderMenu, &p.InFooterMenu, &p.MenuOrder, &p.IsSystem, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt, &p.OGImage)
	return p, err
}

var pageSorts = map[string]string{
	"title": "lower(title)", "slug": "slug", "status": "status", "menu_order": "menu_order",
	"created_at": "created_at", "updated_at": "updated_at", "published_at": "published_at",
}

// PageRepository stores CMS pages.
type PageRepository struct {
	pool *pgxpool.Pool
}

// NewPageRepository returns a PageRepository.
func NewPageRepository(pool *pgxpool.Pool) *PageRepository {
	return &PageRepository{pool: pool}
}

// List returns one page of live pages without their blocks.
func (r *PageRepository) List(ctx context.Context, q ListQuery) (Page[model.Page], error) {
	w := &Where{}
	w.Add("deleted_at IS NULL")
	if q.Q != "" {
		w.Add("(title ILIKE %[1]s OR slug ILIKE %[1]s)", LikePattern(q.Q))
	}
	if v, ok := q.Filters["status"]; ok {
		w.Add("status = %s", v)
	}
	if v, ok := q.Filters["is_system"]; ok {
		w.Add("is_system = %s", v)
	}
	from := " FROM pages" + w.SQL()
	order := Order(q.Sort, q.Desc, pageSorts, "is_system DESC, menu_order, lower(title)")
	limit := " LIMIT " + strconv.Itoa(q.PerPage) + " OFFSET " + strconv.Itoa((q.Page-1)*q.PerPage)

	var page Page[model.Page]
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		rows, err := r.pool.Query(gctx, "SELECT "+pageListColumns+from+order+limit, w.Args()...)
		if err != nil {
			return fmt.Errorf("list pages: %w", err)
		}
		items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Page, error) { return scanPage(row) })
		if err != nil {
			return fmt.Errorf("scan pages: %w", err)
		}
		page.Items = items
		return nil
	})
	g.Go(func() error {
		if err := r.pool.QueryRow(gctx, "SELECT count(*)"+from, w.Args()...).Scan(&page.Total); err != nil {
			return fmt.Errorf("count pages: %w", err)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return Page[model.Page]{}, err
	}
	return page, nil
}

// Get returns a live page by ID.
func (r *PageRepository) Get(ctx context.Context, id uuid.UUID) (model.Page, error) {
	p, err := scanPage(r.pool.QueryRow(ctx, `SELECT `+pageColumns+` FROM pages WHERE id = $1 AND deleted_at IS NULL`, id))
	if err != nil {
		return p, fmt.Errorf("get page: %w", notFound(err))
	}
	return p, nil
}

// SlugTaken reports whether another live page uses slug.
func (r *PageRepository) SlugTaken(ctx context.Context, slug string, exclude uuid.UUID) (bool, error) {
	return SlugTaken(ctx, r.pool, "pages", slug, exclude)
}

// Insert stores a new page and mirrors its menu flags into the menus.
func (r *PageRepository) Insert(ctx context.Context, p model.Page) (model.Page, error) {
	var created model.Page
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		created, err = scanPage(tx.QueryRow(ctx, `INSERT INTO pages
			(title, slug, intro_html, blocks, seo_title, meta_description, og_image_file_id, status, in_header_menu, in_footer_menu, menu_order, published_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING `+pageColumns,
			p.Title, p.Slug, p.IntroHTML, blocksArg(p.Blocks), p.SEOTitle, p.MetaDescription, p.OGImageFileID, p.Status,
			p.InHeaderMenu, p.InFooterMenu, p.MenuOrder, p.PublishedAt))
		if err != nil {
			return fmt.Errorf("insert page: %w", duplicate(err))
		}
		return syncPageMenus(ctx, tx, created)
	})
	return created, err
}

// Update overwrites a live page; redirectFrom, when set, is the old path that must 301 to the new one.
func (r *PageRepository) Update(ctx context.Context, id uuid.UUID, p model.Page, redirectFrom string) (model.Page, error) {
	var updated model.Page
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		updated, err = scanPage(tx.QueryRow(ctx, `UPDATE pages SET title = $2, slug = $3, intro_html = $4, blocks = $5, seo_title = $6,
			meta_description = $7, og_image_file_id = $8, status = $9, in_header_menu = $10, in_footer_menu = $11, menu_order = $12, published_at = $13
			WHERE id = $1 AND deleted_at IS NULL RETURNING `+pageColumns,
			id, p.Title, p.Slug, p.IntroHTML, blocksArg(p.Blocks), p.SEOTitle, p.MetaDescription, p.OGImageFileID, p.Status,
			p.InHeaderMenu, p.InFooterMenu, p.MenuOrder, p.PublishedAt))
		if err != nil {
			return fmt.Errorf("update page: %w", duplicate(notFound(err)))
		}
		if redirectFrom != "" {
			if err := SaveRedirect(ctx, tx, redirectFrom, updated.Path()); err != nil {
				return err
			}
		}
		return syncPageMenus(ctx, tx, updated)
	})
	return updated, err
}

// SoftDelete removes a non-system page and the menu items pointing at it.
func (r *PageRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return InTx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE pages SET deleted_at = now(), in_header_menu = false, in_footer_menu = false
			WHERE id = $1 AND deleted_at IS NULL AND NOT is_system`, id)
		if err != nil {
			return fmt.Errorf("delete page: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		if _, err := tx.Exec(ctx, `DELETE FROM menu_items WHERE page_id = $1`, id); err != nil {
			return fmt.Errorf("delete page menu items: %w", err)
		}
		return nil
	})
}

// ReorderMenuOrder sets menu_order to the position of each page in ids.
func (r *PageRepository) ReorderMenuOrder(ctx context.Context, ids []uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE pages t SET menu_order = o.pos * 10
		FROM unnest($1::uuid[]) WITH ORDINALITY AS o(id, pos) WHERE t.id = o.id AND t.deleted_at IS NULL`, ids)
	if err != nil {
		return fmt.Errorf("reorder pages: %w", err)
	}
	return nil
}

func blocksArg(blocks []model.Block) []model.Block {
	if blocks == nil {
		return []model.Block{}
	}
	return blocks
}

func syncPageMenus(ctx context.Context, tx pgx.Tx, p model.Page) error {
	for menu, listed := range map[string]bool{model.MenuHeader: p.InHeaderMenu, model.MenuFooter: p.InFooterMenu} {
		if !listed {
			if _, err := tx.Exec(ctx, `DELETE FROM menu_items WHERE menu = $1 AND page_id = $2`, menu, p.ID); err != nil {
				return fmt.Errorf("remove page from %s menu: %w", menu, err)
			}
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO menu_items (menu, label, page_id, sort_order)
			SELECT $1::text, $3::text, $2::uuid, CASE WHEN $4::int > 0 THEN $4::int
				ELSE coalesce((SELECT max(sort_order) FROM menu_items WHERE menu = $1 AND parent_id IS NULL), 0) + 10 END
			WHERE NOT EXISTS (SELECT 1 FROM menu_items WHERE menu = $1 AND page_id = $2)`, menu, p.ID, p.Title, p.MenuOrder)
		if err != nil {
			return fmt.Errorf("add page to %s menu: %w", menu, err)
		}
	}
	return nil
}

var errUnknownMenu = errors.New("unknown menu")

func menuFlagColumn(menu string) (string, error) {
	switch menu {
	case model.MenuHeader:
		return "in_header_menu", nil
	case model.MenuFooter:
		return "in_footer_menu", nil
	}
	return "", fmt.Errorf("%w: %q", errUnknownMenu, strings.TrimSpace(menu))
}
