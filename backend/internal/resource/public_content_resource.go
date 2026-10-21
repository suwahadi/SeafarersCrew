package resource

import (
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/video"
)

// PublicNewsCard is an article in a listing.
type PublicNewsCard struct {
	Title       string       `json:"title"`
	Slug        string       `json:"slug"`
	Excerpt     string       `json:"excerpt"`
	PublishedAt time.Time    `json:"published_at"`
	Thumbnail   *PublicImage `json:"thumbnail"`
}

// NewPublicNewsCard returns the article card mapper.
func NewPublicNewsCard(urls MediaURLer) func(model.News) PublicNewsCard {
	return func(n model.News) PublicNewsCard {
		card := PublicNewsCard{Title: n.Title, Slug: n.Slug, Excerpt: n.Excerpt, Thumbnail: NewPublicImage(n.Thumbnail, urls, n.Title)}
		if n.PublishedAt != nil {
			card.PublishedAt = n.PublishedAt.UTC()
		}
		return card
	}
}

// PublicNewsArticle is the payload of GET /news/{slug}.
type PublicNewsArticle struct {
	PublicNewsCard
	BodyHTML        string           `json:"body_html"`
	SEOTitle        string           `json:"seo_title"`
	MetaDescription string           `json:"meta_description"`
	UpdatedAt       time.Time        `json:"updated_at"`
	Related         []PublicNewsCard `json:"related"`
}

// NewPublicNewsArticle maps an article with its related cards.
func NewPublicNewsArticle(d model.NewsDetailData, urls MediaURLer) PublicNewsArticle {
	card := NewPublicNewsCard(urls)
	return PublicNewsArticle{
		PublicNewsCard: card(d.Article), BodyHTML: d.Article.BodyHTML, SEOTitle: d.Article.SEOTitle,
		MetaDescription: d.Article.MetaDescription, UpdatedAt: d.Article.UpdatedAt.UTC(), Related: MapAll(d.Related, card),
	}
}

// PublicGalleryImage is a gallery photo.
type PublicGalleryImage struct {
	ID      uuid.UUID    `json:"id"`
	Caption string       `json:"caption"`
	Image   *PublicImage `json:"image"`
}

// NewPublicGalleryImage returns the gallery image mapper.
func NewPublicGalleryImage(urls MediaURLer) func(model.GalleryImage) PublicGalleryImage {
	return func(g model.GalleryImage) PublicGalleryImage {
		return PublicGalleryImage{ID: g.ID, Caption: g.Caption, Image: NewPublicImage(g.Image, urls, g.AltText)}
	}
}

// PublicHeroSlide is one hero slide.
type PublicHeroSlide struct {
	ID          uuid.UUID    `json:"id"`
	Heading     string       `json:"heading"`
	Subheading  string       `json:"subheading"`
	ButtonLabel string       `json:"button_label"`
	ButtonLink  string       `json:"button_link"`
	Image       *PublicImage `json:"image"`
	MobileImage *PublicImage `json:"mobile_image"`
}

// PublicHero is the hero slider.
type PublicHero struct {
	AutoplaySeconds int               `json:"autoplay_seconds"`
	Slides          []PublicHeroSlide `json:"slides"`
}

// PublicStat is one counter.
type PublicStat struct {
	ID     uuid.UUID `json:"id"`
	Number int       `json:"number"`
	Suffix string    `json:"suffix"`
	Label  string    `json:"label"`
}

// PublicTestimonial is one quote.
type PublicTestimonial struct {
	ID       uuid.UUID    `json:"id"`
	Name     string       `json:"name"`
	Position string       `json:"position"`
	Photo    *PublicImage `json:"photo"`
	Quote    string       `json:"quote"`
	Rating   int          `json:"rating"`
}

// PublicClientLogo is one client.
type PublicClientLogo struct {
	ID   uuid.UUID    `json:"id"`
	Name string       `json:"name"`
	URL  string       `json:"url"`
	Logo *PublicImage `json:"logo"`
}

// PublicPageHead is the SEO and intro part of a page.
type PublicPageHead struct {
	Title           string       `json:"title"`
	SEOTitle        string       `json:"seo_title"`
	MetaDescription string       `json:"meta_description"`
	OGImage         *PublicImage `json:"og_image"`
	IntroHTML       string       `json:"intro_html"`
}

// PublicHome is the payload of GET /home.
type PublicHome struct {
	Page         *PublicPageHead      `json:"page"`
	Sections     []string             `json:"sections"`
	Hero         PublicHero           `json:"hero"`
	Stats        []PublicStat         `json:"stats"`
	News         []PublicNewsCard     `json:"news"`
	Vacancies    []PublicVacancyCard  `json:"vacancies"`
	Gallery      []PublicGalleryImage `json:"gallery"`
	Testimonials []PublicTestimonial  `json:"testimonials"`
	ClientLogos  []PublicClientLogo   `json:"client_logos"`
}

func newPublicPageHead(p model.Page, urls MediaURLer) PublicPageHead {
	return PublicPageHead{
		Title: p.Title, SEOTitle: p.SEOTitle, MetaDescription: p.MetaDescription,
		OGImage: NewPublicImage(p.OGImage, urls, p.Title), IntroHTML: p.IntroHTML,
	}
}

// NewPublicHome maps the home page data.
func NewPublicHome(d model.HomeData, urls MediaURLer) PublicHome {
	out := PublicHome{
		Sections: d.Sections,
		Hero: PublicHero{AutoplaySeconds: d.Autoplay, Slides: MapAll(d.Slides, func(s model.HeroSlide) PublicHeroSlide {
			return PublicHeroSlide{
				ID: s.ID, Heading: s.Heading, Subheading: s.Subheading, ButtonLabel: s.ButtonLabel, ButtonLink: s.ButtonLink,
				Image: NewPublicImage(s.Image, urls, s.Heading), MobileImage: NewPublicImage(s.MobileImage, urls, s.Heading),
			}
		})},
		Stats: MapAll(d.Stats, func(s model.StatsItem) PublicStat {
			return PublicStat{ID: s.ID, Number: s.Number, Suffix: s.Suffix, Label: s.Label}
		}),
		News:      MapAll(d.News, NewPublicNewsCard(urls)),
		Vacancies: NewPublicVacancyCards(d.Vacancies),
		Gallery:   MapAll(d.Gallery, NewPublicGalleryImage(urls)),
		Testimonials: MapAll(d.Testimonials, func(t model.Testimonial) PublicTestimonial {
			return PublicTestimonial{ID: t.ID, Name: t.Name, Position: t.Position, Photo: NewPublicImage(t.Photo, urls, t.Name), Quote: t.Quote, Rating: t.Rating}
		}),
		ClientLogos: MapAll(d.Logos, func(c model.ClientLogo) PublicClientLogo {
			return PublicClientLogo{ID: c.ID, Name: c.Name, URL: c.URL, Logo: NewPublicImage(c.Logo, urls, c.Name)}
		}),
	}
	if out.Sections == nil {
		out.Sections = []string{}
	}
	if d.Page != nil {
		head := newPublicPageHead(*d.Page, urls)
		out.Page = &head
	}
	return out
}

// PublicBlock is a visible page block resolved for rendering.
type PublicBlock struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Settings any    `json:"settings"`
}

// PublicPage is the payload of GET /pages/{slug}.
type PublicPage struct {
	PublicPageHead
	Slug        string        `json:"slug"`
	Path        string        `json:"path"`
	SystemKey   *string       `json:"system_key"`
	Blocks      []PublicBlock `json:"blocks"`
	PublishedAt *time.Time    `json:"published_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// NewPublicPage maps a published page and resolves its blocks.
func NewPublicPage(d model.PageData, urls MediaURLer) PublicPage {
	p := d.Page
	r := blockResolver{data: d, urls: urls}
	blocks := make([]PublicBlock, 0, len(p.Blocks))
	for _, b := range p.Blocks {
		if settings, ok := r.resolve(b); ok {
			blocks = append(blocks, PublicBlock{ID: b.ID, Type: b.Type, Settings: settings})
		}
	}
	return PublicPage{
		PublicPageHead: newPublicPageHead(p, urls), Slug: p.Slug, Path: p.Path(), SystemKey: p.SystemKey, Blocks: blocks,
		PublishedAt: utcPtr(p.PublishedAt), UpdatedAt: p.UpdatedAt.UTC(),
	}
}

type blockResolver struct {
	data model.PageData
	urls MediaURLer
}

func (r blockResolver) image(id *uuid.UUID, alt string) *PublicImage {
	if id == nil {
		return nil
	}
	img, ok := r.data.Images[*id]
	if !ok {
		return nil
	}
	return NewPublicImage(&img, r.urls, alt)
}

type publicImageTextSettings struct {
	Image         *PublicImage `json:"image"`
	Heading       string       `json:"heading"`
	TextHTML      string       `json:"text_html"`
	ButtonLabel   string       `json:"button_label"`
	ButtonLink    string       `json:"button_link"`
	ImagePosition string       `json:"image_position"`
}

type publicGallerySettings struct {
	Images   []PublicGalleryImage `json:"images"`
	Columns  int                  `json:"columns"`
	Lightbox bool                 `json:"lightbox"`
}

type publicVideoSettings struct {
	Provider     string `json:"provider"`
	VideoID      string `json:"video_id"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	EmbedURL     string `json:"embed_url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

type publicCTASettings struct {
	Heading         string       `json:"heading"`
	Text            string       `json:"text"`
	ButtonLabel     string       `json:"button_label"`
	ButtonLink      string       `json:"button_link"`
	BackgroundImage *PublicImage `json:"background_image"`
	BackgroundColor string       `json:"background_color"`
}

type publicCard struct {
	Icon  string       `json:"icon"`
	Image *PublicImage `json:"image"`
	Title string       `json:"title"`
	Text  string       `json:"text"`
	Link  string       `json:"link"`
}

type publicCardGridSettings struct {
	Columns int          `json:"columns"`
	Cards   []publicCard `json:"cards"`
}

func (r blockResolver) resolve(b model.Block) (any, bool) {
	switch v := b.Settings.(type) {
	case *model.RichTextSettings:
		return v, true
	case *model.ImageTextSettings:
		return publicImageTextSettings{
			Image: r.image(v.ImageFileID, v.Heading), Heading: v.Heading, TextHTML: v.TextHTML,
			ButtonLabel: v.ButtonLabel, ButtonLink: v.ButtonLink, ImagePosition: v.ImagePosition,
		}, true
	case *model.GallerySettings:
		return r.gallery(v)
	case *model.VideoSettings:
		ref := video.Ref{Provider: v.Provider, ID: v.VideoID}
		return publicVideoSettings{Provider: v.Provider, VideoID: v.VideoID, Title: v.Title, URL: v.URL, EmbedURL: ref.EmbedURL(), ThumbnailURL: ref.ThumbnailURL()}, true
	case *model.CTASettings:
		return publicCTASettings{
			Heading: v.Heading, Text: v.Text, ButtonLabel: v.ButtonLabel, ButtonLink: v.ButtonLink,
			BackgroundImage: r.image(v.BackgroundImageFileID, v.Heading), BackgroundColor: v.BackgroundColor,
		}, true
	case *model.StatsSettings, *model.FAQSettings:
		return v, true
	case *model.CardGridSettings:
		return publicCardGridSettings{Columns: v.Columns, Cards: MapAll(v.Cards, func(c model.Card) publicCard {
			return publicCard{Icon: c.Icon, Image: r.image(c.ImageFileID, c.Title), Title: c.Title, Text: c.Text, Link: c.Link}
		})}, true
	}
	return nil, false
}

func (r blockResolver) gallery(v *model.GallerySettings) (any, bool) {
	images := make([]PublicGalleryImage, 0, len(v.ImageIDs))
	mapper := NewPublicGalleryImage(r.urls)
	for _, id := range v.ImageIDs {
		if g, ok := r.data.Gallery[id]; ok {
			images = append(images, mapper(g))
		}
	}
	if len(images) == 0 {
		return nil, false
	}
	return publicGallerySettings{Images: images, Columns: v.Columns, Lightbox: v.Lightbox}, true
}

// PublicSitemapEntry is one public URL.
type PublicSitemapEntry struct {
	Path    string    `json:"path"`
	LastMod time.Time `json:"lastmod"`
}

// PublicSitemap is the payload of GET /sitemap-data.
type PublicSitemap struct {
	Pages       []PublicSitemapEntry `json:"pages"`
	SystemPages []PublicSitemapEntry `json:"system_pages"`
	News        []PublicSitemapEntry `json:"news"`
	Vacancies   []PublicSitemapEntry `json:"vacancies"`
}

// NewPublicSitemap maps the sitemap data.
func NewPublicSitemap(d model.SitemapData) PublicSitemap {
	entry := func(e model.SitemapEntry) PublicSitemapEntry {
		return PublicSitemapEntry{Path: e.Path, LastMod: e.LastMod.UTC()}
	}
	return PublicSitemap{
		Pages: MapAll(d.Pages, entry), SystemPages: MapAll(d.System, entry), News: MapAll(d.News, entry), Vacancies: MapAll(d.Vacancies, entry),
	}
}
