package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
)

// Sizes of the public listings.
const (
	HomeNewsCount    = 3
	HomeVacancyCount = 4
	RelatedNewsCount = 3
	MaxNewsPerPage   = 48
	defaultNewsPage  = 9
)

// PublicContent reads published content.
type PublicContent interface {
	PublishedPage(ctx context.Context, slug string) (model.Page, error)
	SystemPage(ctx context.Context, key string) (model.Page, error)
	GalleryImagesByID(ctx context.Context, ids []uuid.UUID) ([]model.GalleryImage, error)
	LiveNews(ctx context.Context, now time.Time, page, perPage int) (repository.Page[model.News], error)
	LatestNews(ctx context.Context, now time.Time, limit int, exclude uuid.UUID) ([]model.News, error)
	LiveNewsBySlug(ctx context.Context, now time.Time, slug string) (model.News, error)
	OpenVacancies(ctx context.Context, today time.Time, limit int) ([]model.Vacancy, error)
	SitemapPages(ctx context.Context) (pages, system []model.SitemapEntry, err error)
	SitemapNews(ctx context.Context, now time.Time) ([]model.SitemapEntry, error)
	SitemapVacancies(ctx context.Context, today time.Time) ([]model.SitemapEntry, error)
}

// PublicImages loads public image data by file ID.
type PublicImages interface {
	Images(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]model.Image, error)
}

// PublicLists groups the active-row listers the public site reads.
type PublicLists struct {
	Slides       ActiveLister[model.HeroSlide]
	Stats        ActiveLister[model.StatsItem]
	Testimonials ActiveLister[model.Testimonial]
	Logos        ActiveLister[model.ClientLogo]
	Gallery      ActiveLister[model.GalleryImage]
	Languages    ActiveLister[model.Language]
}

// PublicService assembles the data behind the public API.
type PublicService struct {
	settings SettingsReader
	content  PublicContent
	menus    MenuStore
	images   PublicImages
	lists    PublicLists
	now      func() time.Time
}

// NewPublicService builds a PublicService.
func NewPublicService(settings SettingsReader, content PublicContent, menus MenuStore, images PublicImages, lists PublicLists, now func() time.Time) *PublicService {
	return &PublicService{settings: settings, content: content, menus: menus, images: images, lists: lists, now: now}
}

var siteSettingGroups = []string{"general", "company", "contact", "branding", "seo", "website", "security"}

// Site returns settings, menus and languages for the public site shell.
func (s *PublicService) Site(ctx context.Context) (model.SiteData, error) {
	var (
		data      = model.SiteData{Settings: map[string]map[string]any{}}
		groups    = make([]map[string]any, len(siteSettingGroups))
		menuItems []model.MenuItem
	)
	g, gctx := errgroup.WithContext(ctx)
	for i, name := range siteSettingGroups {
		g.Go(func() error {
			values, err := s.settings.Values(gctx, name)
			groups[i] = values
			return err
		})
	}
	g.Go(func() error {
		var err error
		menuItems, err = s.menus.Items(gctx)
		return err
	})
	g.Go(func() error {
		var err error
		data.Languages, err = s.lists.Languages.Active(gctx)
		return err
	})
	if err := g.Wait(); err != nil {
		return model.SiteData{}, err
	}
	for i, name := range siteSettingGroups {
		data.Settings[name] = groups[i]
	}
	data.Menus = SplitMenus(menuItems)
	images, err := s.images.Images(ctx, siteImageIDs(data.Settings))
	if err != nil {
		return model.SiteData{}, err
	}
	data.Images = images
	return data, nil
}

// Favicon returns the website favicon chosen in Settings, or nil when none is set.
func (s *PublicService) Favicon(ctx context.Context) (*model.Image, error) {
	company, err := s.settings.Values(ctx, "company")
	if err != nil {
		return nil, err
	}
	id, ok := SettingFileID(company, "favicon_file_id")
	if !ok {
		return nil, nil
	}
	images, err := s.images.Images(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	img, found := images[id]
	if !found {
		return nil, nil
	}
	return &img, nil
}

func siteImageIDs(settings map[string]map[string]any) []uuid.UUID {
	var ids []uuid.UUID
	for _, ref := range [][2]string{{"company", "logo_light_file_id"}, {"company", "logo_dark_file_id"}, {"company", "favicon_file_id"}, {"seo", "og_image_file_id"}} {
		if id, ok := SettingFileID(settings[ref[0]], ref[1]); ok && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids
}

// SettingFileID reads a file ID setting.
func SettingFileID(values map[string]any, key string) (uuid.UUID, bool) {
	raw, _ := values[key].(string)
	id, err := uuid.Parse(raw)
	return id, err == nil
}

// Home returns every section of the home page, loading independent sections in parallel.
func (s *PublicService) Home(ctx context.Context) (model.HomeData, error) {
	var (
		data    model.HomeData
		website map[string]any
		now     = s.now().UTC()
	)
	loc, err := s.location(ctx)
	if err != nil {
		return model.HomeData{}, err
	}
	today := datetime.Today(now, loc)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		website, err = s.settings.Values(gctx, "website")
		return err
	})
	g.Go(func() error {
		var err error
		data.Vacancies, err = s.content.OpenVacancies(gctx, today, HomeVacancyCount)
		return err
	})
	g.Go(func() error {
		page, err := s.content.SystemPage(gctx, model.SystemHome)
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		if err == nil {
			data.Page = &page
		}
		return err
	})
	g.Go(func() error {
		var err error
		data.Slides, err = s.lists.Slides.Active(gctx)
		return err
	})
	g.Go(func() error {
		var err error
		data.Stats, err = s.lists.Stats.Active(gctx)
		return err
	})
	g.Go(func() error {
		var err error
		data.Testimonials, err = s.lists.Testimonials.Active(gctx)
		return err
	})
	g.Go(func() error {
		var err error
		data.Logos, err = s.lists.Logos.Active(gctx)
		return err
	})
	g.Go(func() error {
		all, err := s.lists.Gallery.Active(gctx)
		for _, img := range all {
			if img.ShowOnHome {
				data.Gallery = append(data.Gallery, img)
			}
		}
		return err
	})
	g.Go(func() error {
		var err error
		data.News, err = s.content.LatestNews(gctx, now, HomeNewsCount, uuid.Nil)
		return err
	})
	if err := g.Wait(); err != nil {
		return model.HomeData{}, err
	}
	data.Autoplay = settingInt(website, "hero_autoplay_seconds", 6)
	data.Sections = settingStrings(website, "home_sections")
	return data, nil
}

// Page returns a published page with the visible blocks and the records they reference.
func (s *PublicService) Page(ctx context.Context, slug string) (model.PageData, error) {
	page, err := s.content.PublishedPage(ctx, slug)
	if errors.Is(err, repository.ErrNotFound) {
		return model.PageData{}, apperr.NotFound("Page")
	}
	if err != nil {
		return model.PageData{}, err
	}
	page.Blocks = visibleBlocks(page)
	var galleryIDs []uuid.UUID
	for _, b := range page.Blocks {
		if v, ok := b.Settings.(*model.GallerySettings); ok {
			galleryIDs = append(galleryIDs, v.ImageIDs...)
		}
	}
	data := model.PageData{Page: page}
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		data.Images, err = s.images.Images(gctx, pageImageIDs(page))
		return err
	})
	g.Go(func() error {
		images, err := s.content.GalleryImagesByID(gctx, galleryIDs)
		data.Gallery = make(map[uuid.UUID]model.GalleryImage, len(images))
		for _, img := range images {
			data.Gallery[img.ID] = img
		}
		return err
	})
	if err := g.Wait(); err != nil {
		return model.PageData{}, err
	}
	return data, nil
}

func visibleBlocks(p model.Page) []model.Block {
	if p.IsSystem {
		return nil
	}
	out := make([]model.Block, 0, len(p.Blocks))
	for _, b := range p.Blocks {
		if !b.Hidden {
			out = append(out, b)
		}
	}
	return out
}

// NewsList returns one page of visible articles; perPage 0 uses the website setting.
func (s *PublicService) NewsList(ctx context.Context, page, perPage int) (model.NewsPageData, error) {
	if perPage <= 0 {
		website, err := s.settings.Values(ctx, "website")
		if err != nil {
			return model.NewsPageData{}, err
		}
		perPage = settingInt(website, "news_per_page", defaultNewsPage)
	}
	perPage = min(max(perPage, 1), MaxNewsPerPage)
	page = max(page, 1)
	res, err := s.content.LiveNews(ctx, s.now().UTC(), page, perPage)
	if err != nil {
		return model.NewsPageData{}, err
	}
	return model.NewsPageData{Articles: res.Items, Page: page, PerPage: perPage, Total: res.Total}, nil
}

// NewsDetail returns a visible article with the latest other articles.
func (s *PublicService) NewsDetail(ctx context.Context, slug string) (model.NewsDetailData, error) {
	now := s.now().UTC()
	article, err := s.content.LiveNewsBySlug(ctx, now, slug)
	if errors.Is(err, repository.ErrNotFound) {
		return model.NewsDetailData{}, apperr.NotFound("Article")
	}
	if err != nil {
		return model.NewsDetailData{}, err
	}
	related, err := s.content.LatestNews(ctx, now, RelatedNewsCount, article.ID)
	if err != nil {
		return model.NewsDetailData{}, err
	}
	return model.NewsDetailData{Article: article, Related: related}, nil
}

// Gallery returns every active gallery image in display order.
func (s *PublicService) Gallery(ctx context.Context) ([]model.GalleryImage, error) {
	return s.lists.Gallery.Active(ctx)
}

// Sitemap lists every public URL with its last modification time.
func (s *PublicService) Sitemap(ctx context.Context) (model.SitemapData, error) {
	var (
		data model.SitemapData
		now  = s.now().UTC()
	)
	loc, err := s.location(ctx)
	if err != nil {
		return model.SitemapData{}, err
	}
	today := datetime.Today(now, loc)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		data.Pages, data.System, err = s.content.SitemapPages(gctx)
		return err
	})
	g.Go(func() error {
		var err error
		data.News, err = s.content.SitemapNews(gctx, now)
		return err
	})
	g.Go(func() error {
		var err error
		data.Vacancies, err = s.content.SitemapVacancies(gctx, today)
		return err
	})
	if err := g.Wait(); err != nil {
		return model.SitemapData{}, err
	}
	return data, nil
}

func (s *PublicService) location(ctx context.Context) (*time.Location, error) {
	general, err := s.settings.Values(ctx, "general")
	if err != nil {
		return nil, err
	}
	tz, _ := general["timezone"].(string)
	if loc, err := time.LoadLocation(tz); err == nil && tz != "" {
		return loc, nil
	}
	loc, err := time.LoadLocation(datetime.DefaultTimezone)
	if err != nil {
		return nil, fmt.Errorf("load default timezone: %w", err)
	}
	return loc, nil
}

func settingInt(values map[string]any, key string, fallback int) int {
	if n, ok := values[key].(int); ok && n > 0 {
		return n
	}
	return fallback
}

func settingStrings(values map[string]any, key string) []string {
	list, _ := values[key].([]string)
	return list
}
