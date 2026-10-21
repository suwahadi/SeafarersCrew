package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// MenuStore persists the header and footer menus.
type MenuStore interface {
	Items(ctx context.Context) ([]model.MenuItem, error)
	Replace(ctx context.Context, menu string, items []model.MenuItem) error
}

// PageExistence tells which page IDs are live.
type PageExistence interface {
	PagesExist(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error)
}

// MenuService reads and replaces menus.
type MenuService struct {
	store MenuStore
	pages PageExistence
	site  SiteChanger
}

// NewMenuService builds a MenuService.
func NewMenuService(store MenuStore, pages PageExistence, site SiteChanger) *MenuService {
	return &MenuService{store: store, pages: pages, site: site}
}

// Get returns both menu trees.
func (s *MenuService) Get(ctx context.Context) (model.Menus, error) {
	rows, err := s.store.Items(ctx)
	if err != nil {
		return model.Menus{}, err
	}
	return SplitMenus(rows), nil
}

// SplitMenus nests the rows of both menus.
func SplitMenus(rows []model.MenuItem) model.Menus {
	var header, footer []model.MenuItem
	for _, r := range rows {
		if r.Menu == model.MenuHeader {
			header = append(header, r)
		} else {
			footer = append(footer, r)
		}
	}
	return model.Menus{Header: model.BuildMenuTree(header), Footer: model.BuildMenuTree(footer)}
}

// Replace swaps the named menu for items after checking that every referenced page exists.
func (s *MenuService) Replace(ctx context.Context, menu string, items []model.MenuItem) (model.Menus, error) {
	if menu != model.MenuHeader && menu != model.MenuFooter {
		return model.Menus{}, apperr.NotFound("Menu")
	}
	if err := s.checkPages(ctx, items); err != nil {
		return model.Menus{}, err
	}
	if err := s.store.Replace(ctx, menu, items); err != nil {
		return model.Menus{}, err
	}
	if err := s.site.Changed(ctx, model.TagSite); err != nil {
		return model.Menus{}, err
	}
	return s.Get(ctx)
}

func (s *MenuService) checkPages(ctx context.Context, items []model.MenuItem) error {
	type ref struct {
		field string
		id    uuid.UUID
	}
	var refs []ref
	var ids []uuid.UUID
	collect := func(field string, item model.MenuItem) {
		if item.PageID != nil {
			refs = append(refs, ref{field: field, id: *item.PageID})
			ids = append(ids, *item.PageID)
		}
	}
	for i, item := range items {
		collect(fmt.Sprintf("items.%d.page_id", i), item)
		for j, child := range item.Children {
			collect(fmt.Sprintf("items.%d.children.%d.page_id", i, j), child)
		}
	}
	found, err := s.pages.PagesExist(ctx, ids)
	if err != nil {
		return err
	}
	fields := map[string]string{}
	for _, r := range refs {
		if !found[r.id] {
			fields[r.field] = "The page does not exist."
		}
	}
	if len(fields) > 0 {
		return apperr.Validation(fields)
	}
	return nil
}
