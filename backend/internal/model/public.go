package model

import (
	"time"

	"github.com/google/uuid"
)

// SitemapEntry is one public URL with its last modification time.
type SitemapEntry struct {
	Path    string
	LastMod time.Time
}

// SiteData is everything the public site shell needs.
type SiteData struct {
	Settings  map[string]map[string]any
	Menus     Menus
	Languages []Language
	Images    map[uuid.UUID]Image
}

// HomeData is everything the home page needs.
type HomeData struct {
	Page         *Page
	Sections     []string
	Autoplay     int
	Slides       []HeroSlide
	Stats        []StatsItem
	News         []News
	Vacancies    []Vacancy
	Gallery      []GalleryImage
	Testimonials []Testimonial
	Logos        []ClientLogo
}

// PageData is a published page with the records its blocks reference.
type PageData struct {
	Page    Page
	Gallery map[uuid.UUID]GalleryImage
	Images  map[uuid.UUID]Image
}

// NewsPageData is one page of visible articles.
type NewsPageData struct {
	Articles []News
	Page     int
	PerPage  int
	Total    int
}

// NewsDetailData is a visible article with related articles.
type NewsDetailData struct {
	Article News
	Related []News
}

// SitemapData lists every public URL.
type SitemapData struct {
	Pages     []SitemapEntry
	System    []SitemapEntry
	News      []SitemapEntry
	Vacancies []SitemapEntry
}
