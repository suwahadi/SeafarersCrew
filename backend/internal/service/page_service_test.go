package service

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type memPages struct {
	rows         map[uuid.UUID]model.Page
	lastRedirect string
	reordered    []uuid.UUID
	deleted      []uuid.UUID
}

func newMemPages() *memPages { return &memPages{rows: map[uuid.UUID]model.Page{}} }

func (m *memPages) List(context.Context, repository.ListQuery) (repository.Page[model.Page], error) {
	return repository.Page[model.Page]{}, nil
}

func (m *memPages) Get(_ context.Context, id uuid.UUID) (model.Page, error) {
	p, ok := m.rows[id]
	if !ok {
		return p, repository.ErrNotFound
	}
	return p, nil
}

func (m *memPages) SlugTaken(_ context.Context, slug string, exclude uuid.UUID) (bool, error) {
	for id, p := range m.rows {
		if p.Slug == slug && id != exclude {
			return true, nil
		}
	}
	return false, nil
}

func (m *memPages) Insert(_ context.Context, p model.Page) (model.Page, error) {
	p.ID = uuid.New()
	m.rows[p.ID] = p
	return p, nil
}

func (m *memPages) Update(_ context.Context, id uuid.UUID, p model.Page, redirectFrom string) (model.Page, error) {
	p.ID = id
	p.IsSystem = m.rows[id].IsSystem
	m.rows[id] = p
	m.lastRedirect = redirectFrom
	return p, nil
}

func (m *memPages) SoftDelete(_ context.Context, id uuid.UUID) error {
	m.deleted = append(m.deleted, id)
	delete(m.rows, id)
	return nil
}

func (m *memPages) ReorderMenuOrder(_ context.Context, ids []uuid.UUID) error {
	m.reordered = ids
	return nil
}

type fakePageLookup struct {
	gallery map[uuid.UUID]bool
}

func (f fakePageLookup) GalleryImagesExist(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := map[uuid.UUID]bool{}
	for _, id := range ids {
		out[id] = f.gallery[id]
	}
	return out, nil
}

func (f fakePageLookup) Images(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]model.Image, error) {
	out := map[uuid.UUID]model.Image{}
	for _, id := range ids {
		out[id] = model.Image{ID: id}
	}
	return out, nil
}

type pageFixture struct {
	svc   *PageService
	store *memPages
	files *fakeFileLookup
	site  *siteRecorder
	gal   fakePageLookup
}

func newPageFixture() pageFixture {
	store := newMemPages()
	files := newFakeFiles()
	site := &siteRecorder{}
	gal := fakePageLookup{gallery: map[uuid.UUID]bool{}}
	svc := NewPageService(store, gal, NewPublicFileChecker(files), NewCMSPolicy(), site, fixedNow)
	return pageFixture{svc: svc, store: store, files: files, site: site, gal: gal}
}

func validationFields(t *testing.T, err error) map[string]string {
	t.Helper()
	e, ok := apperr.As(err)
	require.True(t, ok, "expected an application error, got %v", err)
	require.Equal(t, "validation_failed", e.Code)
	return e.Fields
}

func TestPageSlugIsGeneratedFromTheTitleAndSuffixedOnConflict(t *testing.T) {
	f := newPageFixture()
	ctx := context.Background()

	first, err := f.svc.Create(ctx, model.Page{Title: "About Us", Status: model.StatusDraft})
	require.NoError(t, err)
	second, err := f.svc.Create(ctx, model.Page{Title: "About Us", Status: model.StatusDraft})
	require.NoError(t, err)
	third, err := f.svc.Create(ctx, model.Page{Title: "Whatever", Slug: "About  Us!", Status: model.StatusDraft})
	require.NoError(t, err)

	assert.Equal(t, "about-us", first.Page.Slug)
	assert.Equal(t, "about-us-2", second.Page.Slug)
	assert.Equal(t, "about-us-3", third.Page.Slug, "an edited slug is normalized and de-duplicated the same way")
}

func TestPageSlugsOfWebsiteRoutesAreReserved(t *testing.T) {
	f := newPageFixture()
	created, err := f.svc.Create(context.Background(), model.Page{Title: "News", Status: model.StatusDraft})
	require.NoError(t, err)
	assert.Equal(t, "news-2", created.Page.Slug)
	assert.True(t, IsReservedSlug("api"))
	assert.False(t, IsReservedSlug("about-us"))
}

func TestPublishedAtIsSetOnFirstPublishAndKept(t *testing.T) {
	f := newPageFixture()
	ctx := context.Background()
	draft, err := f.svc.Create(ctx, model.Page{Title: "Services", Status: model.StatusDraft})
	require.NoError(t, err)
	assert.Nil(t, draft.Page.PublishedAt)

	published, err := f.svc.Update(ctx, draft.Page.ID, model.Page{Title: "Services", Status: model.StatusPublished})
	require.NoError(t, err)
	require.NotNil(t, published.Page.PublishedAt)
	first := *published.Page.PublishedAt
	assert.Equal(t, fixedNow(), first)

	unpublished, err := f.svc.Update(ctx, draft.Page.ID, model.Page{Title: "Services", Status: model.StatusDraft})
	require.NoError(t, err)
	require.NotNil(t, unpublished.Page.PublishedAt, "the original publication date is remembered")
	assert.Equal(t, first, *unpublished.Page.PublishedAt)
}

func TestChangingThePublishedSlugCreatesARedirect(t *testing.T) {
	cases := []struct {
		name         string
		from, to     string
		newSlug      string
		wantRedirect string
		wantTag      bool
	}{
		{"published to published", model.StatusPublished, model.StatusPublished, "company", "/about-us", true},
		{"draft page has no public url", model.StatusDraft, model.StatusPublished, "company", "", false},
		{"unpublishing keeps no redirect", model.StatusPublished, model.StatusDraft, "company", "", false},
		{"same slug", model.StatusPublished, model.StatusPublished, "about-us", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newPageFixture()
			ctx := context.Background()
			created, err := f.svc.Create(ctx, model.Page{Title: "About Us", Status: c.from})
			require.NoError(t, err)
			_, err = f.svc.Update(ctx, created.Page.ID, model.Page{Title: "About Us", Slug: c.newSlug, Status: c.to})
			require.NoError(t, err)
			assert.Equal(t, c.wantRedirect, f.store.lastRedirect)
			assert.Equal(t, c.wantTag, slices.Contains(f.site.last(), model.TagRedirects))
		})
	}
}

func TestUpdateWithoutASlugKeepsTheCurrentSlug(t *testing.T) {
	f := newPageFixture()
	ctx := context.Background()
	created, err := f.svc.Create(ctx, model.Page{Title: "About Us", Status: model.StatusPublished})
	require.NoError(t, err)
	updated, err := f.svc.Update(ctx, created.Page.ID, model.Page{Title: "A Completely New Title", Status: model.StatusPublished})
	require.NoError(t, err)
	assert.Equal(t, "about-us", updated.Page.Slug)
	assert.Empty(t, f.store.lastRedirect)
}

func TestSystemPagesKeepTheirLayoutSlugAndStatus(t *testing.T) {
	f := newPageFixture()
	ctx := context.Background()
	key := model.SystemContact
	id := uuid.New()
	f.store.rows[id] = model.Page{
		ID: id, Title: "Contact", Slug: "contact", SystemKey: &key, IsSystem: true, Status: model.StatusPublished,
		Blocks: []model.Block{{ID: "x", Type: model.BlockRichText, Settings: &model.RichTextSettings{HTML: "<p>keep</p>"}}},
	}

	updated, err := f.svc.Update(ctx, id, model.Page{
		Title: "Contact Us", Slug: "hacked", Status: model.StatusDraft, SEOTitle: "Reach us", IntroHTML: "<p>Intro</p><script>x</script>",
		Blocks: []model.Block{{ID: "y", Type: model.BlockRichText, Settings: &model.RichTextSettings{HTML: "<p>ignored</p>"}}},
	})
	require.NoError(t, err)
	assert.Equal(t, "contact", updated.Page.Slug)
	assert.Equal(t, model.StatusPublished, updated.Page.Status)
	assert.Equal(t, "Reach us", updated.Page.SEOTitle)
	assert.Equal(t, "<p>Intro</p>", updated.Page.IntroHTML)
	require.Len(t, updated.Page.Blocks, 1)
	assert.Equal(t, "x", updated.Page.Blocks[0].ID, "incoming blocks are ignored for system pages")
	assert.Empty(t, f.store.lastRedirect)

	err = f.svc.Delete(ctx, id)
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, "system_page", e.Code)
	assert.Empty(t, f.store.deleted)
}

func TestPagesAnnounceTheAffectedCacheTags(t *testing.T) {
	f := newPageFixture()
	ctx := context.Background()
	created, err := f.svc.Create(ctx, model.Page{Title: "About Us", Status: model.StatusPublished})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{model.TagPages, model.TagSite, model.TagSitemap}, f.site.last())

	home := model.SystemHome
	id := uuid.New()
	f.store.rows[id] = model.Page{ID: id, Title: "Home", Slug: "home", SystemKey: &home, IsSystem: true, Status: model.StatusPublished}
	_, err = f.svc.Update(ctx, id, model.Page{Title: "Home"})
	require.NoError(t, err)
	assert.Contains(t, f.site.last(), model.TagHome)

	require.NoError(t, f.svc.Delete(ctx, created.Page.ID))
	assert.ElementsMatch(t, []string{model.TagPages, model.TagSite, model.TagSitemap}, f.site.last())
	assert.Equal(t, []uuid.UUID{created.Page.ID}, f.store.deleted)
}

func TestPageHTMLIsSanitizedAndFilesAreChecked(t *testing.T) {
	f := newPageFixture()
	ctx := context.Background()
	image := f.files.add("image/webp", "a.jpg")
	svg := f.files.add("image/svg+xml", "logo.svg")
	galleryID := uuid.New()
	f.gal.gallery[galleryID] = true

	created, err := f.svc.Create(ctx, model.Page{
		Title: "About", Status: model.StatusPublished, IntroHTML: `<p onclick="x()">Hi</p>`, OGImageFileID: &image,
		Blocks: []model.Block{
			{ID: "1", Type: model.BlockRichText, Settings: &model.RichTextSettings{HTML: `<h2>T</h2><script>alert(1)</script><a href="javascript:alert(1)">bad</a><a href="https://example.com">ok</a>`}},
			{ID: "2", Type: model.BlockImageText, Settings: &model.ImageTextSettings{ImageFileID: &image, TextHTML: `<p>t</p><iframe src="x"></iframe>`}},
			{ID: "3", Type: model.BlockFAQ, Settings: &model.FAQSettings{Items: []model.FAQItem{{Question: "Q", AnswerHTML: `<img src="x" onerror="x()">A`}}}},
			{ID: "4", Type: model.BlockGallery, Settings: &model.GallerySettings{ImageIDs: []uuid.UUID{galleryID}, Columns: 3}},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, `<p>Hi</p>`, created.Page.IntroHTML)
	rich := created.Page.Blocks[0].Settings.(*model.RichTextSettings).HTML
	assert.Contains(t, rich, "<h2>T</h2>")
	assert.NotContains(t, rich, "script")
	assert.NotContains(t, rich, "javascript:")
	assert.Contains(t, rich, `href="https://example.com"`)
	assert.NotContains(t, created.Page.Blocks[1].Settings.(*model.ImageTextSettings).TextHTML, "iframe")
	assert.NotContains(t, created.Page.Blocks[2].Settings.(*model.FAQSettings).Items[0].AnswerHTML, "onerror")
	assert.Contains(t, created.Media, image)

	_, err = f.svc.Create(ctx, model.Page{Title: "Bad refs", Status: model.StatusDraft, OGImageFileID: &svg,
		Blocks: []model.Block{
			{ID: "1", Type: model.BlockImageText, Settings: &model.ImageTextSettings{ImageFileID: idPtr(uuid.New())}},
			{ID: "2", Type: model.BlockGallery, Settings: &model.GallerySettings{ImageIDs: []uuid.UUID{galleryID, uuid.New()}, Columns: 3}},
			{ID: "3", Type: model.BlockCardGrid, Settings: &model.CardGridSettings{Columns: 3, Cards: []model.Card{{Title: "x", ImageFileID: idPtr(uuid.New())}}}},
			{ID: "4", Type: model.BlockCTA, Settings: &model.CTASettings{Heading: "h", BackgroundImageFileID: &svg}},
		}})
	fields := validationFields(t, err)
	assert.Contains(t, fields, "og_image_file_id")
	assert.Contains(t, fields, "blocks.0.settings.image_file_id")
	assert.Contains(t, fields, "blocks.1.settings.image_ids.1")
	assert.NotContains(t, fields, "blocks.1.settings.image_ids.0")
	assert.Contains(t, fields, "blocks.2.settings.cards.0.image_file_id")
	assert.Contains(t, fields, "blocks.3.settings.background_image_file_id")
}

func idPtr(id uuid.UUID) *uuid.UUID { return &id }

func TestPageReorderRejectsDuplicatesAndEmptyLists(t *testing.T) {
	f := newPageFixture()
	ctx := context.Background()
	id := uuid.New()
	require.Error(t, f.svc.Reorder(ctx, nil))
	require.Error(t, f.svc.Reorder(ctx, []uuid.UUID{id, id}))
	require.NoError(t, f.svc.Reorder(ctx, []uuid.UUID{id}))
	assert.Equal(t, []uuid.UUID{id}, f.store.reordered)
}

func TestMissingPagesAreReportedAsNotFound(t *testing.T) {
	f := newPageFixture()
	_, err := f.svc.Get(context.Background(), uuid.New())
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, 404, e.Status)
}
