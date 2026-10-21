package request

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// Limits of page content.
const (
	MaxBlocks         = 100
	maxPageIntroBytes = 50_000
)

// Page is the body of POST and PUT /pages.
type Page struct {
	Title           string            `json:"title"`
	Slug            string            `json:"slug"`
	SEOTitle        string            `json:"seo_title"`
	MetaDescription string            `json:"meta_description"`
	OGImageFileID   *uuid.UUID        `json:"og_image_file_id"`
	Status          string            `json:"status"`
	InHeaderMenu    bool              `json:"in_header_menu"`
	InFooterMenu    bool              `json:"in_footer_menu"`
	MenuOrder       int               `json:"menu_order"`
	IntroHTML       string            `json:"intro_html"`
	Blocks          []json.RawMessage `json:"blocks"`

	parsed []model.Block
}

// Validate implements Validator.
func (r *Page) Validate() map[string]string {
	trimAll(&r.Title, &r.Slug, &r.SEOTitle, &r.MetaDescription, &r.Status)
	if r.Status == "" {
		r.Status = model.StatusDraft
	}
	f := FieldErrors{}
	f.Required("title", r.Title, 200)
	f.MaxLen("slug", r.Slug, 120)
	f.MaxLen("seo_title", r.SEOTitle, 120)
	f.MaxLen("meta_description", r.MetaDescription, 320)
	f.OneOf("status", r.Status, model.StatusDraft, model.StatusPublished)
	if r.MenuOrder < 0 || r.MenuOrder > 10000 {
		f.set("menu_order", "The menu order must be between 0 and 10000.")
	}
	if len(r.IntroHTML) > maxPageIntroBytes {
		f.set("intro_html", fmt.Sprintf("The intro may not exceed %d characters.", maxPageIntroBytes))
	}
	r.parsed = ParseBlocks(r.Blocks, f)
	return f
}

// Model converts the request to a model.
func (r *Page) Model() model.Page {
	return model.Page{
		Title: r.Title, Slug: r.Slug, SEOTitle: r.SEOTitle, MetaDescription: r.MetaDescription, OGImageFileID: r.OGImageFileID,
		Status: r.Status, InHeaderMenu: r.InHeaderMenu, InFooterMenu: r.InFooterMenu, MenuOrder: r.MenuOrder,
		IntroHTML: r.IntroHTML, Blocks: r.parsed,
	}
}
