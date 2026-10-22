package seeders

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type systemPageSeed struct {
	key, title, intro string
	header, footer    bool
}

var systemPages = []systemPageSeed{
	{"home", "Home", "", true, false},
	{"news", "News", "", true, true},
	{"career", "Career", "", true, true},
	{"gallery", "Gallery", "", true, false},
	{"contact", "Contact", "<p>Get in touch with our crewing team.</p>", true, true},
}

func seedSystemPages(ctx context.Context, tx pgx.Tx) error {
	b := &pgx.Batch{}
	for i, p := range systemPages {
		b.Queue(`INSERT INTO pages (title, slug, system_key, is_system, intro_html, status, published_at, in_header_menu, in_footer_menu, menu_order)
			SELECT $1, $2, $2, true, $3, 'published', now(), $4, $5, $6
			WHERE NOT EXISTS (SELECT 1 FROM pages WHERE system_key = $2)
			  AND NOT EXISTS (SELECT 1 FROM pages WHERE slug = $2 AND deleted_at IS NULL)`,
			p.title, p.key, p.intro, p.header, p.footer, (i+1)*10)
	}
	return execBatch(ctx, tx, b)
}

func seedMenus(ctx context.Context, tx pgx.Tx) error {
	var existing int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM menu_items`).Scan(&existing); err != nil {
		return fmt.Errorf("count menu items: %w", err)
	}
	if existing > 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `INSERT INTO menu_items (menu, label, page_id, sort_order)
		SELECT m.menu, p.title, p.id, p.menu_order
		FROM pages p
		JOIN LATERAL (SELECT 'header' AS menu WHERE p.in_header_menu UNION ALL SELECT 'footer' WHERE p.in_footer_menu) m ON true
		WHERE p.is_system AND p.deleted_at IS NULL`)
	if err != nil {
		return fmt.Errorf("insert default menu items: %w", err)
	}
	return nil
}
