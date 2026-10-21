package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type memMenus struct {
	rows     []model.MenuItem
	replaced string
}

func (m *memMenus) Items(context.Context) ([]model.MenuItem, error) { return m.rows, nil }

func (m *memMenus) Replace(_ context.Context, menu string, items []model.MenuItem) error {
	m.replaced = menu
	var rows []model.MenuItem
	for _, r := range m.rows {
		if r.Menu != menu {
			rows = append(rows, r)
		}
	}
	for _, it := range items {
		parent := model.MenuItem{ID: uuid.New(), Menu: menu, Label: it.Label, PageID: it.PageID, URL: it.URL}
		rows = append(rows, parent)
		for _, c := range it.Children {
			rows = append(rows, model.MenuItem{ID: uuid.New(), Menu: menu, Label: c.Label, URL: c.URL, ParentID: &parent.ID})
		}
	}
	m.rows = rows
	return nil
}

type fakePages map[uuid.UUID]bool

func (f fakePages) PagesExist(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := map[uuid.UUID]bool{}
	for _, id := range ids {
		out[id] = f[id]
	}
	return out, nil
}

func TestMenuReplaceChecksPagesAndReturnsBothTrees(t *testing.T) {
	store := &memMenus{}
	site := &siteRecorder{}
	about := uuid.New()
	svc := NewMenuService(store, fakePages{about: true}, site)
	ctx := context.Background()

	menus, err := svc.Replace(ctx, model.MenuHeader, []model.MenuItem{
		{Label: "About", PageID: &about, Children: []model.MenuItem{{Label: "Safety", URL: "/safety"}}},
		{Label: "Contact", URL: "/contact"},
	})
	require.NoError(t, err)
	require.Len(t, menus.Header, 2)
	require.Len(t, menus.Header[0].Children, 1)
	assert.Empty(t, menus.Footer)
	assert.Equal(t, []string{model.TagSite}, site.last())
}

func TestMenuReplaceRejectsMissingPagesWithNestedFieldKeys(t *testing.T) {
	store := &memMenus{}
	site := &siteRecorder{}
	known, gone := uuid.New(), uuid.New()
	svc := NewMenuService(store, fakePages{known: true}, site)

	_, err := svc.Replace(context.Background(), model.MenuFooter, []model.MenuItem{
		{Label: "Gone", PageID: &gone},
		{Label: "Parent", URL: "/p", Children: []model.MenuItem{{Label: "Known", PageID: &known}, {Label: "Gone too", PageID: &gone}}},
	})
	fields := validationFields(t, err)
	assert.Contains(t, fields, "items.0.page_id")
	assert.Contains(t, fields, "items.1.children.1.page_id")
	assert.NotContains(t, fields, "items.1.children.0.page_id")
	assert.Empty(t, store.replaced, "nothing is stored when validation fails")
	assert.Zero(t, site.count())
}

func TestMenuReplaceRejectsUnknownMenus(t *testing.T) {
	svc := NewMenuService(&memMenus{}, fakePages{}, &siteRecorder{})
	_, err := svc.Replace(context.Background(), "sidebar", nil)
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, 404, e.Status)
}

func TestSplitMenusKeepsTheOrderOfRows(t *testing.T) {
	parent := uuid.New()
	menus := SplitMenus([]model.MenuItem{
		{ID: parent, Menu: model.MenuHeader, Label: "A"},
		{ID: uuid.New(), Menu: model.MenuHeader, Label: "A1", ParentID: &parent},
		{ID: uuid.New(), Menu: model.MenuHeader, Label: "A2", ParentID: &parent},
		{ID: uuid.New(), Menu: model.MenuFooter, Label: "F"},
	})
	require.Len(t, menus.Header, 1)
	assert.Equal(t, []string{"A1", "A2"}, []string{menus.Header[0].Children[0].Label, menus.Header[0].Children[1].Label})
	require.Len(t, menus.Footer, 1)
}
