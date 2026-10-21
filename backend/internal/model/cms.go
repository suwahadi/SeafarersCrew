package model

import (
	"time"

	"github.com/google/uuid"
)

// Content statuses of pages and news.
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

// HeroSlide is one slide of the home hero slider.
type HeroSlide struct {
	MasterBase
	ImageFileID       uuid.UUID
	MobileImageFileID *uuid.UUID
	Heading           string
	Subheading        string
	ButtonLabel       string
	ButtonLink        string
	Image             *Image
	MobileImage       *Image
}

// GalleryImage is one gallery photo.
type GalleryImage struct {
	MasterBase
	ImageFileID uuid.UUID
	Caption     string
	AltText     string
	ShowOnHome  bool
	Image       *Image
}

// Testimonial is a client or crew quote.
type Testimonial struct {
	MasterBase
	Name        string
	Position    string
	PhotoFileID *uuid.UUID
	Quote       string
	Rating      int
	Photo       *Image
}

// ClientLogo is a client shown in the logo strip.
type ClientLogo struct {
	MasterBase
	Name       string
	LogoFileID uuid.UUID
	URL        string
	Logo       *Image
}

// StatsItem is one counter of the home page stats band.
type StatsItem struct {
	MasterBase
	Number int
	Suffix string
	Label  string
}

// Language is an entry of the public language switcher.
type Language struct {
	MasterBase
	Name       string
	Code       string
	FlagFileID *uuid.UUID
	FlagCode   string
	IsSource   bool
	Flag       *Image
}

// PublicFileInfo is what the CMS needs to know about a referenced public file.
type PublicFileInfo struct {
	ID           uuid.UUID
	MimeType     string
	OriginalName string
}

// News is a news article.
type News struct {
	ID              uuid.UUID
	Title           string
	Slug            string
	Excerpt         string
	BodyHTML        string
	ThumbnailFileID *uuid.UUID
	PublishedAt     *time.Time
	Status          string
	SEOTitle        string
	MetaDescription string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Thumbnail       *Image
}

// Live reports whether the article is visible on the public site at now.
func (n News) Live(now time.Time) bool {
	return n.Status == StatusPublished && n.PublishedAt != nil && !n.PublishedAt.After(now)
}

// Menu names.
const (
	MenuHeader = "header"
	MenuFooter = "footer"
)

// MenuItem is one link of the header or footer menu.
type MenuItem struct {
	ID           uuid.UUID
	Menu         string
	Label        string
	PageID       *uuid.UUID
	URL          string
	ParentID     *uuid.UUID
	OpenInNewTab bool
	SortOrder    int
	Page         *MenuPage
	Children     []MenuItem
}

// MenuPage is the page a menu item points to.
type MenuPage struct {
	Title     string
	Slug      string
	SystemKey *string
	Status    string
}

// Menus holds both menu trees.
type Menus struct {
	Header []MenuItem
	Footer []MenuItem
}

// BuildMenuTree nests children below their parents, keeping the order of rows.
func BuildMenuTree(rows []MenuItem) []MenuItem {
	children := map[uuid.UUID][]MenuItem{}
	var roots []MenuItem
	for _, r := range rows {
		if r.ParentID != nil {
			children[*r.ParentID] = append(children[*r.ParentID], r)
			continue
		}
		roots = append(roots, r)
	}
	for i := range roots {
		roots[i].Children = children[roots[i].ID]
	}
	return roots
}
