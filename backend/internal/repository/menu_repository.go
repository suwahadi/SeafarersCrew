package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// MenuRepository stores the header and footer menus.
type MenuRepository struct {
	pool *pgxpool.Pool
}

// NewMenuRepository returns a MenuRepository.
func NewMenuRepository(pool *pgxpool.Pool) *MenuRepository {
	return &MenuRepository{pool: pool}
}

// Items returns every menu item of both menus in display order, with the page each one points to.
func (r *MenuRepository) Items(ctx context.Context) ([]model.MenuItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT m.id, m.menu, m.label, m.page_id, coalesce(m.url, ''), m.parent_id, m.open_in_new_tab, m.sort_order,
		p.title, p.slug, p.system_key, p.status
		FROM menu_items m LEFT JOIN pages p ON p.id = m.page_id AND p.deleted_at IS NULL
		ORDER BY m.menu, m.sort_order, m.created_at, m.id`)
	if err != nil {
		return nil, fmt.Errorf("list menu items: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.MenuItem, error) {
		var (
			m      model.MenuItem
			title  *string
			slug   *string
			system *string
			status *string
		)
		if err := row.Scan(&m.ID, &m.Menu, &m.Label, &m.PageID, &m.URL, &m.ParentID, &m.OpenInNewTab, &m.SortOrder, &title, &slug, &system, &status); err != nil {
			return m, err
		}
		if title != nil && slug != nil && status != nil {
			m.Page = &model.MenuPage{Title: *title, Slug: *slug, SystemKey: system, Status: *status}
		}
		return m, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan menu items: %w", err)
	}
	return items, nil
}

// Replace swaps the whole menu for items in one transaction and mirrors the result into the pages' menu flags.
func (r *MenuRepository) Replace(ctx context.Context, menu string, items []model.MenuItem) error {
	column, err := menuFlagColumn(menu)
	if err != nil {
		return err
	}
	return InTx(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM menu_items WHERE menu = $1`, menu); err != nil {
			return fmt.Errorf("clear %s menu: %w", menu, err)
		}
		for i, item := range items {
			parentID, err := insertMenuItem(ctx, tx, menu, item, nil, (i+1)*10)
			if err != nil {
				return err
			}
			for j, child := range item.Children {
				if _, err := insertMenuItem(ctx, tx, menu, child, &parentID, (j+1)*10); err != nil {
					return err
				}
			}
		}
		_, err := tx.Exec(ctx, `UPDATE pages SET `+column+` = EXISTS (SELECT 1 FROM menu_items m WHERE m.page_id = pages.id AND m.menu = $1)
			WHERE deleted_at IS NULL AND `+column+` IS DISTINCT FROM EXISTS (SELECT 1 FROM menu_items m WHERE m.page_id = pages.id AND m.menu = $1)`, menu)
		if err != nil {
			return fmt.Errorf("sync %s menu flags: %w", menu, err)
		}
		return nil
	})
}

func insertMenuItem(ctx context.Context, tx pgx.Tx, menu string, item model.MenuItem, parent *string, order int) (string, error) {
	var url *string
	if item.PageID == nil {
		url = &item.URL
	}
	var id string
	err := tx.QueryRow(ctx, `INSERT INTO menu_items (menu, label, page_id, url, parent_id, open_in_new_tab, sort_order)
		VALUES ($1, $2, $3, $4, $5::uuid, $6, $7) RETURNING id::text`,
		menu, item.Label, item.PageID, url, parent, item.OpenInNewTab, order).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert menu item: %w", err)
	}
	return id, nil
}
