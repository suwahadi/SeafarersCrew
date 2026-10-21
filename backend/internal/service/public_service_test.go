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
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type fakeSettings map[string]map[string]any

func (f fakeSettings) Values(_ context.Context, group string) (map[string]any, error) {
	return f[group], nil
}

type fakeContent struct {
	pages        map[string]model.Page
	system       map[string]model.Page
	gallery      []model.GalleryImage
	news         []model.News
	vacancies    []model.Vacancy
	todayArg     time.Time
	newsLimit    int
	liveNewsPage [2]int
}

func (f *fakeContent) PublishedPage(_ context.Context, slug string) (model.Page, error) {
	p, ok := f.pages[slug]
	if !ok {
		return p, repository.ErrNotFound
	}
	return p, nil
}

func (f *fakeContent) SystemPage(_ context.Context, key string) (model.Page, error) {
	p, ok := f.system[key]
	if !ok {
		return p, repository.ErrNotFound
	}
	return p, nil
}

func (f *fakeContent) GalleryImagesByID(_ context.Context, ids []uuid.UUID) ([]model.GalleryImage, error) {
	var out []model.GalleryImage
	for _, g := range f.gallery {
		for _, id := range ids {
			if g.ID == id {
				out = append(out, g)
			}
		}
	}
	return out, nil
}

func (f *fakeContent) LiveNews(_ context.Context, _ time.Time, page, perPage int) (repository.Page[model.News], error) {
	f.liveNewsPage = [2]int{page, perPage}
	return repository.Page[model.News]{Items: f.news, Total: len(f.news)}, nil
}

func (f *fakeContent) LatestNews(_ context.Context, _ time.Time, limit int, exclude uuid.UUID) ([]model.News, error) {
	f.newsLimit = limit
	var out []model.News
	for _, n := range f.news {
		if n.ID != exclude {
			out = append(out, n)
		}
	}
	return out, nil
}

func (f *fakeContent) LiveNewsBySlug(_ context.Context, _ time.Time, slug string) (model.News, error) {
	for _, n := range f.news {
		if n.Slug == slug {
			return n, nil
		}
	}
	return model.News{}, repository.ErrNotFound
}

func (f *fakeContent) OpenVacancies(_ context.Context, today time.Time, _ int) ([]model.Vacancy, error) {
	f.todayArg = today
	return f.vacancies, nil
}

func (f *fakeContent) SitemapPages(context.Context) ([]model.SitemapEntry, []model.SitemapEntry, error) {
	return []model.SitemapEntry{{Path: "/about"}}, []model.SitemapEntry{{Path: "/"}}, nil
}

func (f *fakeContent) SitemapNews(context.Context, time.Time) ([]model.SitemapEntry, error) {
	return []model.SitemapEntry{{Path: "/news/a"}}, nil
}

func (f *fakeContent) SitemapVacancies(_ context.Context, today time.Time) ([]model.SitemapEntry, error) {
	f.todayArg = today
	return []model.SitemapEntry{{Path: "/career/a"}}, nil
}

type fakeActive[T any] []T

func (f fakeActive[T]) Active(context.Context) ([]T, error) { return f, nil }

type fakePublicImages map[uuid.UUID]model.Image

func (f fakePublicImages) Images(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]model.Image, error) {
	out := map[uuid.UUID]model.Image{}
	for _, id := range ids {
		if img, ok := f[id]; ok {
			out[id] = img
		}
	}
	return out, nil
}

func newPublicFixture(content *fakeContent, settings fakeSettings, now time.Time) *PublicService {
	lists := PublicLists{
		Slides:       fakeActive[model.HeroSlide]{{Heading: "Slide"}},
		Stats:        fakeActive[model.StatsItem]{{Label: "Crew"}},
		Testimonials: fakeActive[model.Testimonial]{{Name: "Ari"}},
		Logos:        fakeActive[model.ClientLogo]{{Name: "Acme"}},
		Gallery:      fakeActive[model.GalleryImage]{{AltText: "home", ShowOnHome: true}, {AltText: "other"}},
		Languages:    fakeActive[model.Language]{{Code: "en"}},
	}
	return NewPublicService(settings, content, &memMenus{}, fakePublicImages{}, lists, func() time.Time { return now })
}

func TestHomeAssemblesEverySectionAndFiltersGalleryHighlights(t *testing.T) {
	content := &fakeContent{
		system:    map[string]model.Page{model.SystemHome: {Title: "Home", IntroHTML: "<p>Hi</p>"}},
		news:      []model.News{{ID: uuid.New(), Slug: "a"}, {ID: uuid.New(), Slug: "b"}},
		vacancies: []model.Vacancy{{Slug: "master"}},
	}
	svc := newPublicFixture(content, fakeSettings{
		"general": {"timezone": "Asia/Jakarta"},
		"website": {"hero_autoplay_seconds": 8, "home_sections": []string{"hero", "news"}},
	}, time.Date(2026, 10, 21, 18, 0, 0, 0, time.UTC))

	home, err := svc.Home(context.Background())
	require.NoError(t, err)
	require.NotNil(t, home.Page)
	assert.Equal(t, "Home", home.Page.Title)
	assert.Equal(t, 8, home.Autoplay)
	assert.Equal(t, []string{"hero", "news"}, home.Sections)
	assert.Len(t, home.Slides, 1)
	assert.Len(t, home.Stats, 1)
	assert.Len(t, home.Testimonials, 1)
	assert.Len(t, home.Logos, 1)
	assert.Len(t, home.News, 2)
	assert.Equal(t, HomeNewsCount, content.newsLimit)
	assert.Len(t, home.Vacancies, 1)
	require.Len(t, home.Gallery, 1)
	assert.Equal(t, "home", home.Gallery[0].AltText, "only images flagged for the home page are highlighted")
}

func TestVacanciesAreFilteredByTodayInTheConfiguredTimezone(t *testing.T) {
	cases := []struct {
		name string
		tz   string
		want time.Time
	}{
		{"jakarta is already tomorrow", "Asia/Jakarta", time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)},
		{"utc is still today", "UTC", time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)},
		{"invalid falls back to the default zone", "Mars/Base", time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			content := &fakeContent{}
			svc := newPublicFixture(content, fakeSettings{"general": {"timezone": c.tz}, "website": {}}, time.Date(2026, 10, 21, 18, 0, 0, 0, time.UTC))
			_, err := svc.Home(context.Background())
			require.NoError(t, err)
			assert.Equal(t, c.want, content.todayArg)

			_, err = svc.Sitemap(context.Background())
			require.NoError(t, err)
			assert.Equal(t, c.want, content.todayArg)
		})
	}
}

func TestHomeWithoutAHomePageStillRenders(t *testing.T) {
	svc := newPublicFixture(&fakeContent{}, fakeSettings{"general": {}, "website": {}}, fixedNow())
	home, err := svc.Home(context.Background())
	require.NoError(t, err)
	assert.Nil(t, home.Page)
	assert.Equal(t, 6, home.Autoplay)
}

func TestPublicPageHidesHiddenBlocksAndIgnoresSystemPageBlocks(t *testing.T) {
	galleryID := uuid.New()
	rich := func(id string, hidden bool) model.Block {
		return model.Block{ID: id, Type: model.BlockRichText, Hidden: hidden, Settings: &model.RichTextSettings{HTML: "<p>x</p>"}}
	}
	content := &fakeContent{
		pages: map[string]model.Page{
			"about": {Slug: "about", Blocks: []model.Block{
				rich("a", false), rich("b", true),
				{ID: "g", Type: model.BlockGallery, Settings: &model.GallerySettings{ImageIDs: []uuid.UUID{galleryID}, Columns: 3}},
			}},
			"contact": {Slug: "contact", IsSystem: true, Blocks: []model.Block{rich("s", false)}},
		},
		gallery: []model.GalleryImage{{MasterBase: model.MasterBase{ID: galleryID}, AltText: "deck"}},
	}
	svc := newPublicFixture(content, fakeSettings{}, fixedNow())

	data, err := svc.Page(context.Background(), "about")
	require.NoError(t, err)
	require.Len(t, data.Page.Blocks, 2)
	assert.Equal(t, "a", data.Page.Blocks[0].ID)
	assert.Contains(t, data.Gallery, galleryID)

	system, err := svc.Page(context.Background(), "contact")
	require.NoError(t, err)
	assert.Empty(t, system.Page.Blocks)

	_, err = svc.Page(context.Background(), "draft-page")
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, 404, e.Status)
}

func TestNewsListUsesTheSettingAndCapsPerPage(t *testing.T) {
	cases := []struct {
		name          string
		page, perPage int
		wantPage      [2]int
	}{
		{"setting default", 0, 0, [2]int{1, 12}},
		{"explicit", 2, 6, [2]int{2, 6}},
		{"capped at 48", 1, 500, [2]int{1, 48}},
		{"negative page", -3, 5, [2]int{1, 5}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			content := &fakeContent{news: []model.News{{Slug: "a"}}}
			svc := newPublicFixture(content, fakeSettings{"website": {"news_per_page": 12}}, fixedNow())
			data, err := svc.NewsList(context.Background(), c.page, c.perPage)
			require.NoError(t, err)
			assert.Equal(t, c.wantPage, content.liveNewsPage)
			assert.Equal(t, c.wantPage[1], data.PerPage)
			assert.Equal(t, 1, data.Total)
		})
	}
}

func TestNewsDetailReturnsThreeRelatedArticlesExcludingItself(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	content := &fakeContent{news: []model.News{{ID: a, Slug: "a"}, {ID: b, Slug: "b"}}}
	svc := newPublicFixture(content, fakeSettings{}, fixedNow())

	data, err := svc.NewsDetail(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "a", data.Article.Slug)
	require.Len(t, data.Related, 1)
	assert.Equal(t, "b", data.Related[0].Slug)
	assert.Equal(t, RelatedNewsCount, content.newsLimit)

	_, err = svc.NewsDetail(context.Background(), "missing")
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, 404, e.Status)
}

func TestSiteCollectsSettingsMenusLanguagesAndLogoImages(t *testing.T) {
	logo := uuid.New()
	images := fakePublicImages{logo: {ID: logo, StorageKey: "public/logo/2026/10/x.svg"}}
	settings := fakeSettings{}
	for _, g := range siteSettingGroups {
		settings[g] = map[string]any{}
	}
	settings["company"]["logo_light_file_id"] = logo.String()
	settings["company"]["logo_dark_file_id"] = "not-a-uuid"
	lists := PublicLists{Languages: fakeActive[model.Language]{{Code: "en"}, {Code: "id"}}}
	svc := NewPublicService(settings, &fakeContent{}, &memMenus{rows: []model.MenuItem{{Menu: model.MenuHeader, Label: "Home", URL: "/"}}}, images, lists, fixedNow)

	data, err := svc.Site(context.Background())
	require.NoError(t, err)
	assert.Len(t, data.Languages, 2)
	require.Len(t, data.Menus.Header, 1)
	assert.Contains(t, data.Images, logo)
	assert.Len(t, data.Settings, len(siteSettingGroups))
}

func TestSitemapCombinesPagesNewsAndVacancies(t *testing.T) {
	svc := newPublicFixture(&fakeContent{}, fakeSettings{"general": {}}, fixedNow())
	data, err := svc.Sitemap(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/about", data.Pages[0].Path)
	assert.Equal(t, "/", data.System[0].Path)
	assert.Equal(t, "/news/a", data.News[0].Path)
	assert.Equal(t, "/career/a", data.Vacancies[0].Path)
}
