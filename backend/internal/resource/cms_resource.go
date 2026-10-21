package resource

import (
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

// AdminImage is a public image as shown in the admin UI.
type AdminImage struct {
	FileID   uuid.UUID `json:"file_id"`
	URL      string    `json:"url"`
	ThumbURL string    `json:"thumb_url"`
	MimeType string    `json:"mime_type"`
	Width    int       `json:"width"`
	Height   int       `json:"height"`
}

// NewAdminImage maps a public image, or nil when there is none.
func NewAdminImage(img *model.Image, urls MediaURLer) *AdminImage {
	if img == nil {
		return nil
	}
	url := urls.URL(img.StorageKey)
	if url == "" {
		return nil
	}
	thumb := url
	smallest := 0
	for _, v := range img.Variants {
		if u := urls.URL(v.StorageKey); u != "" && (smallest == 0 || v.Width < smallest) {
			thumb, smallest = u, v.Width
		}
	}
	return &AdminImage{FileID: img.ID, URL: url, ThumbURL: thumb, MimeType: img.MimeType, Width: img.Width, Height: img.Height}
}

// HeroSlide is the API shape of a hero slide.
type HeroSlide struct {
	MasterBase
	ImageFileID       uuid.UUID   `json:"image_file_id"`
	MobileImageFileID *uuid.UUID  `json:"mobile_image_file_id"`
	Heading           string      `json:"heading"`
	Subheading        string      `json:"subheading"`
	ButtonLabel       string      `json:"button_label"`
	ButtonLink        string      `json:"button_link"`
	Image             *AdminImage `json:"image"`
	MobileImage       *AdminImage `json:"mobile_image"`
}

// NewHeroSlide returns the hero slide mapper.
func NewHeroSlide(urls MediaURLer) func(model.HeroSlide) HeroSlide {
	return func(s model.HeroSlide) HeroSlide {
		return HeroSlide{
			MasterBase: masterBase(s.MasterBase), ImageFileID: s.ImageFileID, MobileImageFileID: s.MobileImageFileID,
			Heading: s.Heading, Subheading: s.Subheading, ButtonLabel: s.ButtonLabel, ButtonLink: s.ButtonLink,
			Image: NewAdminImage(s.Image, urls), MobileImage: NewAdminImage(s.MobileImage, urls),
		}
	}
}

// GalleryImage is the API shape of a gallery image.
type GalleryImage struct {
	MasterBase
	ImageFileID uuid.UUID   `json:"image_file_id"`
	Caption     string      `json:"caption"`
	AltText     string      `json:"alt_text"`
	ShowOnHome  bool        `json:"show_on_home"`
	Image       *AdminImage `json:"image"`
}

// NewGalleryImage returns the gallery image mapper.
func NewGalleryImage(urls MediaURLer) func(model.GalleryImage) GalleryImage {
	return func(g model.GalleryImage) GalleryImage {
		return GalleryImage{
			MasterBase: masterBase(g.MasterBase), ImageFileID: g.ImageFileID, Caption: g.Caption, AltText: g.AltText,
			ShowOnHome: g.ShowOnHome, Image: NewAdminImage(g.Image, urls),
		}
	}
}

// Testimonial is the API shape of a testimonial.
type Testimonial struct {
	MasterBase
	Name        string      `json:"name"`
	Position    string      `json:"position"`
	PhotoFileID *uuid.UUID  `json:"photo_file_id"`
	Quote       string      `json:"quote"`
	Rating      int         `json:"rating"`
	Photo       *AdminImage `json:"photo"`
}

// NewTestimonial returns the testimonial mapper.
func NewTestimonial(urls MediaURLer) func(model.Testimonial) Testimonial {
	return func(t model.Testimonial) Testimonial {
		return Testimonial{
			MasterBase: masterBase(t.MasterBase), Name: t.Name, Position: t.Position, PhotoFileID: t.PhotoFileID,
			Quote: t.Quote, Rating: t.Rating, Photo: NewAdminImage(t.Photo, urls),
		}
	}
}

// ClientLogo is the API shape of a client logo.
type ClientLogo struct {
	MasterBase
	Name       string      `json:"name"`
	LogoFileID uuid.UUID   `json:"logo_file_id"`
	URL        string      `json:"url"`
	Logo       *AdminImage `json:"logo"`
}

// NewClientLogo returns the client logo mapper.
func NewClientLogo(urls MediaURLer) func(model.ClientLogo) ClientLogo {
	return func(c model.ClientLogo) ClientLogo {
		return ClientLogo{
			MasterBase: masterBase(c.MasterBase), Name: c.Name, LogoFileID: c.LogoFileID, URL: c.URL, Logo: NewAdminImage(c.Logo, urls),
		}
	}
}

// StatsItem is the API shape of a stats item.
type StatsItem struct {
	MasterBase
	Number int    `json:"number"`
	Suffix string `json:"suffix"`
	Label  string `json:"label"`
}

// NewStatsItem maps a stats item.
func NewStatsItem(s model.StatsItem) StatsItem {
	return StatsItem{MasterBase: masterBase(s.MasterBase), Number: s.Number, Suffix: s.Suffix, Label: s.Label}
}

// Language is the API shape of a language.
type Language struct {
	MasterBase
	Name       string      `json:"name"`
	Code       string      `json:"code"`
	FlagFileID *uuid.UUID  `json:"flag_file_id"`
	FlagCode   string      `json:"flag_code"`
	IsSource   bool        `json:"is_source"`
	Flag       *AdminImage `json:"flag"`
}

// NewLanguage returns the language mapper.
func NewLanguage(urls MediaURLer) func(model.Language) Language {
	return func(l model.Language) Language {
		return Language{
			MasterBase: masterBase(l.MasterBase), Name: l.Name, Code: l.Code, FlagFileID: l.FlagFileID, FlagCode: l.FlagCode,
			IsSource: l.IsSource, Flag: NewAdminImage(l.Flag, urls),
		}
	}
}

// PageListItem is the API shape of a page in a list.
type PageListItem struct {
	ID           uuid.UUID  `json:"id"`
	Title        string     `json:"title"`
	Slug         string     `json:"slug"`
	Path         string     `json:"path"`
	SystemKey    *string    `json:"system_key"`
	IsSystem     bool       `json:"is_system"`
	Status       string     `json:"status"`
	InHeaderMenu bool       `json:"in_header_menu"`
	InFooterMenu bool       `json:"in_footer_menu"`
	MenuOrder    int        `json:"menu_order"`
	PublishedAt  *time.Time `json:"published_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// NewPageListItem maps a page without its content.
func NewPageListItem(p model.Page) PageListItem {
	return PageListItem{
		ID: p.ID, Title: p.Title, Slug: p.Slug, Path: p.Path(), SystemKey: p.SystemKey, IsSystem: p.IsSystem, Status: p.Status,
		InHeaderMenu: p.InHeaderMenu, InFooterMenu: p.InFooterMenu, MenuOrder: p.MenuOrder,
		PublishedAt: utcPtr(p.PublishedAt), CreatedAt: p.CreatedAt.UTC(), UpdatedAt: p.UpdatedAt.UTC(),
	}
}

// Page is the full API shape of a page.
type Page struct {
	PageListItem
	IntroHTML       string                    `json:"intro_html"`
	SEOTitle        string                    `json:"seo_title"`
	MetaDescription string                    `json:"meta_description"`
	OGImageFileID   *uuid.UUID                `json:"og_image_file_id"`
	OGImage         *AdminImage               `json:"og_image"`
	Blocks          []model.Block             `json:"blocks"`
	Media           map[uuid.UUID]*AdminImage `json:"media"`
}

// NewPage returns the page mapper.
func NewPage(urls MediaURLer) func(service.PageDetail) Page {
	return func(d service.PageDetail) Page {
		p := d.Page
		blocks := p.Blocks
		if blocks == nil {
			blocks = []model.Block{}
		}
		media := make(map[uuid.UUID]*AdminImage, len(d.Media))
		for id, img := range d.Media {
			if mapped := NewAdminImage(&img, urls); mapped != nil {
				media[id] = mapped
			}
		}
		return Page{
			PageListItem: NewPageListItem(p), IntroHTML: p.IntroHTML, SEOTitle: p.SEOTitle, MetaDescription: p.MetaDescription,
			OGImageFileID: p.OGImageFileID, OGImage: NewAdminImage(p.OGImage, urls), Blocks: blocks, Media: media,
		}
	}
}

// NewsListItem is the API shape of an article in a list.
type NewsListItem struct {
	ID              uuid.UUID   `json:"id"`
	Title           string      `json:"title"`
	Slug            string      `json:"slug"`
	Excerpt         string      `json:"excerpt"`
	ThumbnailFileID *uuid.UUID  `json:"thumbnail_file_id"`
	Thumbnail       *AdminImage `json:"thumbnail"`
	PublishedAt     *time.Time  `json:"published_at"`
	Status          string      `json:"status"`
	IsLive          bool        `json:"is_live"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// NewNewsListItem returns the article list mapper; now decides which articles are live.
func NewNewsListItem(urls MediaURLer, now func() time.Time) func(model.News) NewsListItem {
	return func(n model.News) NewsListItem {
		return NewsListItem{
			ID: n.ID, Title: n.Title, Slug: n.Slug, Excerpt: n.Excerpt, ThumbnailFileID: n.ThumbnailFileID,
			Thumbnail: NewAdminImage(n.Thumbnail, urls), PublishedAt: utcPtr(n.PublishedAt), Status: n.Status,
			IsLive: n.Live(now()), CreatedAt: n.CreatedAt.UTC(), UpdatedAt: n.UpdatedAt.UTC(),
		}
	}
}

// News is the full API shape of an article.
type News struct {
	NewsListItem
	BodyHTML        string `json:"body_html"`
	SEOTitle        string `json:"seo_title"`
	MetaDescription string `json:"meta_description"`
}

// NewNews returns the article mapper.
func NewNews(urls MediaURLer, now func() time.Time) func(model.News) News {
	item := NewNewsListItem(urls, now)
	return func(n model.News) News {
		return News{NewsListItem: item(n), BodyHTML: n.BodyHTML, SEOTitle: n.SEOTitle, MetaDescription: n.MetaDescription}
	}
}

// MenuPage is the page a menu item points to.
type MenuPage struct {
	Title     string  `json:"title"`
	Slug      string  `json:"slug"`
	SystemKey *string `json:"system_key"`
	Status    string  `json:"status"`
	Path      string  `json:"path"`
}

// MenuItem is the API shape of a menu item.
type MenuItem struct {
	ID           uuid.UUID  `json:"id"`
	Label        string     `json:"label"`
	PageID       *uuid.UUID `json:"page_id"`
	URL          string     `json:"url"`
	OpenInNewTab bool       `json:"open_in_new_tab"`
	Page         *MenuPage  `json:"page"`
	Children     []MenuItem `json:"children"`
}

// Menus is the payload of GET and PUT /menus.
type Menus struct {
	Header []MenuItem `json:"header"`
	Footer []MenuItem `json:"footer"`
}

// NewMenus maps both menu trees.
func NewMenus(m model.Menus) Menus {
	return Menus{Header: MapAll(m.Header, newMenuItem), Footer: MapAll(m.Footer, newMenuItem)}
}

func newMenuItem(m model.MenuItem) MenuItem {
	item := MenuItem{ID: m.ID, Label: m.Label, PageID: m.PageID, URL: m.URL, OpenInNewTab: m.OpenInNewTab, Children: MapAll(m.Children, newMenuItem)}
	if m.Page != nil {
		path := "/" + m.Page.Slug
		if m.Page.SystemKey != nil {
			path = model.SystemPath(*m.Page.SystemKey)
		}
		item.Page = &MenuPage{Title: m.Page.Title, Slug: m.Page.Slug, SystemKey: m.Page.SystemKey, Status: m.Page.Status, Path: path}
	}
	return item
}

func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
