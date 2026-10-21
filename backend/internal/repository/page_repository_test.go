package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

func samplePage(slug string) model.Page {
	return model.Page{
		Title: "About Us", Slug: slug, Status: model.StatusDraft, IntroHTML: "<p>Hi</p>",
		Blocks: []model.Block{
			{ID: "b1", Type: model.BlockRichText, Settings: &model.RichTextSettings{HTML: "<p>Body</p>"}},
			{ID: "b2", Type: model.BlockGallery, Hidden: true, Settings: &model.GallerySettings{ImageIDs: []uuid.UUID{uuid.New()}, Columns: 3, Lightbox: true}},
		},
	}
}

func menuItemsOf(t *testing.T, menu string) []model.MenuItem {
	t.Helper()
	rows, err := repository.NewMenuRepository(testDB).Items(context.Background())
	require.NoError(t, err)
	var out []model.MenuItem
	for _, r := range rows {
		if r.Menu == menu {
			out = append(out, r)
		}
	}
	return out
}

func TestPageRepositoryStoresBlocksAsTypedJSON(t *testing.T) {
	truncate(t, "pages", "menu_items", "redirects")
	ctx := context.Background()
	repo := repository.NewPageRepository(testDB)

	created, err := repo.Insert(ctx, samplePage("about-us"))
	require.NoError(t, err)
	got, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)

	require.Len(t, got.Blocks, 2)
	rich, ok := got.Blocks[0].Settings.(*model.RichTextSettings)
	require.True(t, ok)
	assert.Equal(t, "<p>Body</p>", rich.HTML)
	gallery, ok := got.Blocks[1].Settings.(*model.GallerySettings)
	require.True(t, ok)
	assert.True(t, got.Blocks[1].Hidden)
	assert.True(t, gallery.Lightbox)
	assert.Equal(t, "/about-us", got.Path())
}

func TestPageRepositoryListFiltersAndOmitsBlocks(t *testing.T) {
	truncate(t, "pages", "menu_items")
	ctx := context.Background()
	repo := repository.NewPageRepository(testDB)
	published := samplePage("about-us")
	published.Status = model.StatusPublished
	_, err := repo.Insert(ctx, published)
	require.NoError(t, err)
	_, err = repo.Insert(ctx, samplePage("services"))
	require.NoError(t, err)

	page, err := repo.List(ctx, listAll(repository.ListQuery{Filters: map[string]any{"status": "published"}}))
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	assert.Empty(t, page.Items[0].Blocks, "list rows carry no blocks")

	page, err = repo.List(ctx, listAll(repository.ListQuery{Q: "serv"}))
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	assert.Equal(t, "services", page.Items[0].Slug)
}

func TestPageRepositoryMirrorsMenuFlagsIntoMenus(t *testing.T) {
	truncate(t, "pages", "menu_items")
	ctx := context.Background()
	repo := repository.NewPageRepository(testDB)
	in := samplePage("about-us")
	in.InHeaderMenu = true
	created, err := repo.Insert(ctx, in)
	require.NoError(t, err)

	header := menuItemsOf(t, model.MenuHeader)
	require.Len(t, header, 1)
	assert.Equal(t, created.ID, *header[0].PageID)
	assert.Empty(t, menuItemsOf(t, model.MenuFooter))

	in.InHeaderMenu, in.InFooterMenu = false, true
	_, err = repo.Update(ctx, created.ID, in, "")
	require.NoError(t, err)
	assert.Empty(t, menuItemsOf(t, model.MenuHeader))
	assert.Len(t, menuItemsOf(t, model.MenuFooter), 1)

	require.NoError(t, repo.SoftDelete(ctx, created.ID))
	assert.Empty(t, menuItemsOf(t, model.MenuFooter), "deleting a page removes its menu items")
	_, err = repo.Get(ctx, created.ID)
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestPageRepositoryUpdateStoresRedirectInTheSameTransaction(t *testing.T) {
	truncate(t, "pages", "menu_items", "redirects")
	ctx := context.Background()
	repo := repository.NewPageRepository(testDB)
	created, err := repo.Insert(ctx, samplePage("about-us"))
	require.NoError(t, err)

	moved := samplePage("company")
	_, err = repo.Update(ctx, created.ID, moved, "/about-us")
	require.NoError(t, err)
	rd, err := repository.NewRedirectRepository(testDB).Resolve(ctx, "/about-us")
	require.NoError(t, err)
	assert.Equal(t, "/company", rd.ToPath)

	other, err := repo.Insert(ctx, samplePage("services"))
	require.NoError(t, err)
	_, err = repo.Update(ctx, other.ID, samplePage("company"), "/services")
	require.ErrorIs(t, err, repository.ErrDuplicate)
	_, err = repository.NewRedirectRepository(testDB).Resolve(ctx, "/services")
	assert.ErrorIs(t, err, repository.ErrNotFound, "a failed update leaves no redirect behind")
}

func TestPageRepositoryRefusesToDeleteSystemPages(t *testing.T) {
	truncate(t, "pages", "menu_items")
	ctx := context.Background()
	id := mustID(t, `INSERT INTO pages (title, slug, system_key, is_system, status) VALUES ('Home', 'home', 'home', true, 'published') RETURNING id`)
	err := repository.NewPageRepository(testDB).SoftDelete(ctx, id)
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestPageRepositoryReorderSetsMenuOrder(t *testing.T) {
	truncate(t, "pages", "menu_items")
	ctx := context.Background()
	repo := repository.NewPageRepository(testDB)
	a, err := repo.Insert(ctx, samplePage("a"))
	require.NoError(t, err)
	b, err := repo.Insert(ctx, samplePage("b"))
	require.NoError(t, err)
	require.NoError(t, repo.ReorderMenuOrder(ctx, []uuid.UUID{b.ID, a.ID}))
	got, err := repo.Get(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, 10, got.MenuOrder)
	got, err = repo.Get(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, 20, got.MenuOrder)
}

func TestMenuRepositoryReplaceIsAtomicAndSyncsPageFlags(t *testing.T) {
	truncate(t, "pages", "menu_items")
	ctx := context.Background()
	pages := repository.NewPageRepository(testDB)
	about, err := pages.Insert(ctx, samplePage("about-us"))
	require.NoError(t, err)
	services, err := pages.Insert(ctx, samplePage("services"))
	require.NoError(t, err)
	menus := repository.NewMenuRepository(testDB)

	err = menus.Replace(ctx, model.MenuHeader, []model.MenuItem{
		{Label: "About", PageID: &about.ID, Children: []model.MenuItem{
			{Label: "Services", PageID: &services.ID},
			{Label: "Safety", URL: "https://example.com/safety", OpenInNewTab: true},
		}},
		{Label: "Contact", URL: "/contact"},
	})
	require.NoError(t, err)

	tree := model.BuildMenuTree(menuItemsOf(t, model.MenuHeader))
	require.Len(t, tree, 2)
	assert.Equal(t, "About", tree[0].Label)
	require.Len(t, tree[0].Children, 2)
	assert.Equal(t, "Services", tree[0].Children[0].Label)
	require.NotNil(t, tree[0].Children[0].Page)
	assert.Equal(t, "services", tree[0].Children[0].Page.Slug)
	assert.True(t, tree[0].Children[1].OpenInNewTab)
	assert.Equal(t, "/contact", tree[1].URL)

	got, err := pages.Get(ctx, services.ID)
	require.NoError(t, err)
	assert.True(t, got.InHeaderMenu)

	require.NoError(t, menus.Replace(ctx, model.MenuHeader, []model.MenuItem{{Label: "About", PageID: &about.ID}}))
	got, err = pages.Get(ctx, services.ID)
	require.NoError(t, err)
	assert.False(t, got.InHeaderMenu)

	unknown := uuid.New()
	err = menus.Replace(ctx, model.MenuHeader, []model.MenuItem{{Label: "Ghost", PageID: &unknown}})
	require.Error(t, err)
	assert.Len(t, menuItemsOf(t, model.MenuHeader), 1, "a failed replace keeps the previous menu")

	require.Error(t, menus.Replace(ctx, "sidebar", nil))
}

func TestNewsRepositoryListsAndRedirects(t *testing.T) {
	truncate(t, "news", "redirects")
	ctx := context.Background()
	repo := repository.NewNewsRepository(testDB)
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	created, err := repo.Insert(ctx, model.News{Title: "Crew change", Slug: "crew-change", Status: model.StatusPublished, PublishedAt: &past, BodyHTML: "<p>Body</p>"})
	require.NoError(t, err)
	_, err = repo.Insert(ctx, model.News{Title: "Draft story", Slug: "draft-story", Status: model.StatusDraft})
	require.NoError(t, err)

	page, err := repo.List(ctx, listAll(repository.ListQuery{Filters: map[string]any{"status": "published"}}))
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	assert.Empty(t, page.Items[0].BodyHTML, "list rows carry no body")
	assert.True(t, page.Items[0].Live(now))

	page, err = repo.List(ctx, listAll(repository.ListQuery{Q: "draft", Sort: "published_at", Desc: true}))
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)

	moved := created
	moved.Slug = "crew-rotation"
	updated, err := repo.Update(ctx, created.ID, moved, "/news/crew-change")
	require.NoError(t, err)
	assert.Equal(t, "crew-rotation", updated.Slug)
	rd, err := repository.NewRedirectRepository(testDB).Resolve(ctx, "/news/crew-change")
	require.NoError(t, err)
	assert.Equal(t, "/news/crew-rotation", rd.ToPath)

	require.NoError(t, repo.SoftDelete(ctx, created.ID))
	assert.ErrorIs(t, repo.SoftDelete(ctx, created.ID), repository.ErrNotFound)
}
