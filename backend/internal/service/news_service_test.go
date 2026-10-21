package service

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

type memNews struct {
	rows         map[uuid.UUID]model.News
	lastRedirect string
}

func newMemNews() *memNews { return &memNews{rows: map[uuid.UUID]model.News{}} }

func (m *memNews) List(context.Context, repository.ListQuery) (repository.Page[model.News], error) {
	return repository.Page[model.News]{}, nil
}

func (m *memNews) Get(_ context.Context, id uuid.UUID) (model.News, error) {
	n, ok := m.rows[id]
	if !ok {
		return n, repository.ErrNotFound
	}
	return n, nil
}

func (m *memNews) SlugTaken(_ context.Context, slug string, exclude uuid.UUID) (bool, error) {
	for id, n := range m.rows {
		if n.Slug == slug && id != exclude {
			return true, nil
		}
	}
	return false, nil
}

func (m *memNews) Insert(_ context.Context, n model.News) (model.News, error) {
	n.ID = uuid.New()
	m.rows[n.ID] = n
	return n, nil
}

func (m *memNews) Update(_ context.Context, id uuid.UUID, n model.News, redirectFrom string) (model.News, error) {
	n.ID = id
	m.rows[id] = n
	m.lastRedirect = redirectFrom
	return n, nil
}

func (m *memNews) SoftDelete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.rows[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.rows, id)
	return nil
}

type newsFixture struct {
	svc   *NewsService
	store *memNews
	files *fakeFileLookup
	site  *siteRecorder
}

func newNewsFixture() newsFixture {
	store := newMemNews()
	files := newFakeFiles()
	site := &siteRecorder{}
	return newsFixture{svc: NewNewsService(store, NewPublicFileChecker(files), NewCMSPolicy(), site, fixedNow), store: store, files: files, site: site}
}

func TestNewsPublishedWithoutADateUsesTheCurrentTime(t *testing.T) {
	f := newNewsFixture()
	created, err := f.svc.Create(context.Background(), model.News{Title: "Crew change", Status: model.StatusPublished})
	require.NoError(t, err)
	require.NotNil(t, created.PublishedAt)
	assert.Equal(t, fixedNow(), *created.PublishedAt)
	assert.Equal(t, "crew-change", created.Slug)
	assert.True(t, created.Live(fixedNow()))
}

func TestNewsWithAFutureDateStaysHiddenUntilThen(t *testing.T) {
	f := newNewsFixture()
	later := fixedNow().Add(48 * time.Hour)
	created, err := f.svc.Create(context.Background(), model.News{Title: "Scheduled", Status: model.StatusPublished, PublishedAt: &later})
	require.NoError(t, err)
	assert.False(t, created.Live(fixedNow()))
	assert.True(t, created.Live(later))
}

func TestNewsDraftsKeepNoPublishDateUnlessGiven(t *testing.T) {
	f := newNewsFixture()
	created, err := f.svc.Create(context.Background(), model.News{Title: "Draft", Status: model.StatusDraft})
	require.NoError(t, err)
	assert.Nil(t, created.PublishedAt)
	assert.False(t, created.Live(fixedNow()))
}

func TestNewsSlugsAreUniqueAndOnlyVisibleArticlesRedirect(t *testing.T) {
	f := newNewsFixture()
	ctx := context.Background()
	past := fixedNow().Add(-time.Hour)
	live, err := f.svc.Create(ctx, model.News{Title: "Crew change", Status: model.StatusPublished, PublishedAt: &past})
	require.NoError(t, err)
	dup, err := f.svc.Create(ctx, model.News{Title: "Crew change", Status: model.StatusDraft})
	require.NoError(t, err)
	assert.Equal(t, "crew-change-2", dup.Slug)

	_, err = f.svc.Update(ctx, live.ID, model.News{Title: "Crew change", Slug: "crew-rotation", Status: model.StatusPublished, PublishedAt: &past})
	require.NoError(t, err)
	assert.Equal(t, "/news/crew-change", f.store.lastRedirect)
	assert.Contains(t, f.site.last(), model.TagRedirects)

	_, err = f.svc.Update(ctx, dup.ID, model.News{Title: "Crew change", Slug: "something-else", Status: model.StatusDraft})
	require.NoError(t, err)
	assert.Empty(t, f.store.lastRedirect, "a draft was never public so it needs no redirect")

	scheduled := fixedNow().Add(time.Hour)
	_, err = f.svc.Update(ctx, live.ID, model.News{Title: "Crew change", Slug: "crew-rotation-2", Status: model.StatusPublished, PublishedAt: &scheduled})
	require.NoError(t, err)
	assert.Equal(t, "/news/crew-rotation", f.store.lastRedirect, "the article was visible before this edit")
}

func TestNewsUpdateRepublishKeepsTheOriginalDate(t *testing.T) {
	f := newNewsFixture()
	ctx := context.Background()
	past := fixedNow().Add(-72 * time.Hour)
	created, err := f.svc.Create(ctx, model.News{Title: "Story", Status: model.StatusPublished, PublishedAt: &past})
	require.NoError(t, err)
	draft, err := f.svc.Update(ctx, created.ID, model.News{Title: "Story", Status: model.StatusDraft})
	require.NoError(t, err)
	assert.Nil(t, draft.PublishedAt)
	again, err := f.svc.Update(ctx, created.ID, model.News{Title: "Story", Status: model.StatusPublished})
	require.NoError(t, err)
	require.NotNil(t, again.PublishedAt)
	assert.Equal(t, fixedNow(), *again.PublishedAt)
}

func TestNewsBodyIsSanitizedAndThumbnailChecked(t *testing.T) {
	f := newNewsFixture()
	ctx := context.Background()
	svg := f.files.add("image/svg+xml", "x.svg")
	_, err := f.svc.Create(ctx, model.News{Title: "Story", ThumbnailFileID: &svg})
	assert.Contains(t, validationFields(t, err), "thumbnail_file_id")

	image := f.files.add("image/webp", "x.jpg")
	created, err := f.svc.Create(ctx, model.News{Title: "Story", ThumbnailFileID: &image, BodyHTML: `<p>Hi <b onmouseover="x()">there</b></p><script>1</script>`})
	require.NoError(t, err)
	assert.Equal(t, `<p>Hi <b>there</b></p>`, created.BodyHTML)
}

func TestNewsWritesAnnounceNewsHomeAndSitemap(t *testing.T) {
	f := newNewsFixture()
	ctx := context.Background()
	created, err := f.svc.Create(ctx, model.News{Title: "Story"})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{model.TagNews, model.TagHome, model.TagSitemap}, f.site.last())
	before := f.site.count()
	require.NoError(t, f.svc.Delete(ctx, created.ID))
	assert.Equal(t, before+1, f.site.count())
	assert.Error(t, f.svc.Delete(ctx, created.ID))
}
