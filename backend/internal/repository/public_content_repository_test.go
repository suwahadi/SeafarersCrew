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

func TestPublicContentShowsOnlyPublishedPages(t *testing.T) {
	truncate(t, "pages", "menu_items")
	ctx := context.Background()
	pages := repository.NewPageRepository(testDB)
	public := repository.NewPublicContentRepository(testDB)
	live := samplePage("about-us")
	live.Status = model.StatusPublished
	_, err := pages.Insert(ctx, live)
	require.NoError(t, err)
	_, err = pages.Insert(ctx, samplePage("secret-draft"))
	require.NoError(t, err)
	exec(t, `INSERT INTO pages (title, slug, system_key, is_system, status) VALUES ('Home', 'home', 'home', true, 'published')`)

	got, err := public.PublishedPage(ctx, "about-us")
	require.NoError(t, err)
	assert.Len(t, got.Blocks, 2)
	_, err = public.PublishedPage(ctx, "secret-draft")
	assert.ErrorIs(t, err, repository.ErrNotFound)
	home, err := public.PublishedPage(ctx, "home")
	require.NoError(t, err)
	assert.True(t, home.IsSystem)
	sys, err := public.SystemPage(ctx, model.SystemHome)
	require.NoError(t, err)
	assert.Equal(t, home.ID, sys.ID)

	normal, system, err := public.SitemapPages(ctx)
	require.NoError(t, err)
	require.Len(t, normal, 1)
	assert.Equal(t, "/about-us", normal[0].Path)
	require.Len(t, system, 1)
	assert.Equal(t, "/", system[0].Path)
}

func TestPublicContentHidesFutureAndDraftNews(t *testing.T) {
	truncate(t, "news")
	ctx := context.Background()
	repo := repository.NewNewsRepository(testDB)
	public := repository.NewPublicContentRepository(testDB)
	now := time.Now().UTC()
	insert := func(slug, status string, at *time.Time) model.News {
		n, err := repo.Insert(ctx, model.News{Title: slug, Slug: slug, Status: status, PublishedAt: at})
		require.NoError(t, err)
		return n
	}
	hour := func(h int) *time.Time { t := now.Add(time.Duration(h) * time.Hour); return &t }
	oldest := insert("oldest", model.StatusPublished, hour(-3))
	insert("newest", model.StatusPublished, hour(-1))
	insert("scheduled", model.StatusPublished, hour(2))
	insert("draft", model.StatusDraft, hour(-2))
	insert("undated", model.StatusPublished, nil)

	page, err := public.LiveNews(ctx, now, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, page.Total)
	require.Len(t, page.Items, 2)
	assert.Equal(t, "newest", page.Items[0].Slug)

	page, err = public.LiveNews(ctx, now, 2, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, page.Total)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "oldest", page.Items[0].Slug)

	later, err := public.LiveNews(ctx, now.Add(3*time.Hour), 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 3, later.Total, "a scheduled article appears once its time has come")

	related, err := public.LatestNews(ctx, now, 3, oldest.ID)
	require.NoError(t, err)
	require.Len(t, related, 1)
	assert.Equal(t, "newest", related[0].Slug)

	_, err = public.LiveNewsBySlug(ctx, now, "scheduled")
	assert.ErrorIs(t, err, repository.ErrNotFound)
	got, err := public.LiveNewsBySlug(ctx, now, "newest")
	require.NoError(t, err)
	assert.Equal(t, "newest", got.Title)

	entries, err := public.SitemapNews(ctx, now)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "/news/newest", entries[0].Path)
}

func TestPublicContentListsOnlyOpenVacanciesNotPastTheirClosingDate(t *testing.T) {
	truncate(t, "vacancies", "ranks", "vessel_types")
	ctx := context.Background()
	rank := mustID(t, `INSERT INTO ranks (name, short_code, department, is_officer) VALUES ('Master', 'MST', 'deck', true) RETURNING id`)
	vt := mustID(t, `INSERT INTO vessel_types (name) VALUES ('AHTS') RETURNING id`)
	today := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	insert := func(pid, slug, status string, closing any, deleted bool) {
		del := any(nil)
		if deleted {
			del = today
		}
		exec(t, `INSERT INTO vacancies (public_id, title, slug, rank_id, vessel_type_id, location, status, closing_date, deleted_at)
			VALUES ($1, $2, $2, $3, $4, 'Batam', $5, $6, $7)`, pid, slug, rank, vt, status, closing, del)
	}
	insert("VACAAAAA1", "open-no-close", "open", nil, false)
	insert("VACAAAAA2", "open-today", "open", today, false)
	insert("VACAAAAA3", "open-expired", "open", today.AddDate(0, 0, -1), false)
	insert("VACAAAAA4", "draft-one", "draft", nil, false)
	insert("VACAAAAA5", "closed-one", "closed", nil, false)
	insert("VACAAAAA6", "open-deleted", "open", nil, true)

	public := repository.NewPublicContentRepository(testDB)
	cards, err := public.OpenVacancies(ctx, today, 4)
	require.NoError(t, err)
	require.Len(t, cards, 2)
	slugs := []string{cards[0].Slug, cards[1].Slug}
	assert.ElementsMatch(t, []string{"open-no-close", "open-today"}, slugs)
	assert.Equal(t, "Master", cards[0].RankName)
	assert.Equal(t, "AHTS", cards[0].VesselTypeName)

	limited, err := public.OpenVacancies(ctx, today, 1)
	require.NoError(t, err)
	assert.Len(t, limited, 1)

	entries, err := public.SitemapVacancies(ctx, today)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Contains(t, entries[0].Path, "/career/open")
}

func TestPublicContentGalleryImagesByIDKeepsRequestedOrderAndSkipsInactive(t *testing.T) {
	truncate(t, "gallery_images", "files")
	ctx := context.Background()
	repo := repository.NewGalleryImageRepository(testDB)
	file := publicFile(t, "image/webp")
	a, err := repo.Insert(ctx, model.GalleryImage{MasterBase: model.MasterBase{IsActive: true}, ImageFileID: file, AltText: "a"})
	require.NoError(t, err)
	b, err := repo.Insert(ctx, model.GalleryImage{MasterBase: model.MasterBase{IsActive: true}, ImageFileID: file, AltText: "b"})
	require.NoError(t, err)
	off, err := repo.Insert(ctx, model.GalleryImage{MasterBase: model.MasterBase{IsActive: false}, ImageFileID: file, AltText: "off"})
	require.NoError(t, err)

	got, err := repository.NewPublicContentRepository(testDB).GalleryImagesByID(ctx, []uuid.UUID{b.ID, off.ID, a.ID, uuid.New()})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "b", got[0].AltText)
	assert.Equal(t, "a", got[1].AltText)
	require.NotNil(t, got[0].Image)

	none, err := repository.NewPublicContentRepository(testDB).GalleryImagesByID(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestCMSLookupReportsLiveRecords(t *testing.T) {
	truncate(t, "pages", "gallery_images", "files", "menu_items")
	ctx := context.Background()
	file := publicFile(t, "image/webp")
	gallery, err := repository.NewGalleryImageRepository(testDB).Insert(ctx, model.GalleryImage{MasterBase: model.MasterBase{IsActive: true}, ImageFileID: file, AltText: "a"})
	require.NoError(t, err)
	page, err := repository.NewPageRepository(testDB).Insert(ctx, samplePage("about-us"))
	require.NoError(t, err)
	lookup := repository.NewCMSLookupRepository(testDB)

	found, err := lookup.GalleryImagesExist(ctx, []uuid.UUID{gallery.ID, uuid.New()})
	require.NoError(t, err)
	assert.Equal(t, map[uuid.UUID]bool{gallery.ID: true}, found)
	found, err = lookup.PagesExist(ctx, []uuid.UUID{page.ID})
	require.NoError(t, err)
	assert.True(t, found[page.ID])
	found, err = lookup.PagesExist(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, found)
}
