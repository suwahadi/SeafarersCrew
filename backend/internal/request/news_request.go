package request

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

const maxNewsBodyBytes = 200_000

// News is the body of POST and PUT /news.
type News struct {
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Excerpt         string     `json:"excerpt"`
	BodyHTML        string     `json:"body_html"`
	ThumbnailFileID *uuid.UUID `json:"thumbnail_file_id"`
	PublishedAt     *time.Time `json:"published_at"`
	Status          string     `json:"status"`
	SEOTitle        string     `json:"seo_title"`
	MetaDescription string     `json:"meta_description"`
}

// Validate implements Validator.
func (r *News) Validate() map[string]string {
	trimAll(&r.Title, &r.Slug, &r.Excerpt, &r.Status, &r.SEOTitle, &r.MetaDescription)
	if r.Status == "" {
		r.Status = model.StatusDraft
	}
	f := FieldErrors{}
	f.Required("title", r.Title, 200)
	f.MaxLen("slug", r.Slug, 120)
	f.MaxLen("excerpt", r.Excerpt, 300)
	f.MaxLen("seo_title", r.SEOTitle, 120)
	f.MaxLen("meta_description", r.MetaDescription, 320)
	f.OneOf("status", r.Status, model.StatusDraft, model.StatusPublished)
	if len(r.BodyHTML) > maxNewsBodyBytes {
		f.set("body_html", fmt.Sprintf("The body may not exceed %d characters.", maxNewsBodyBytes))
	}
	return f
}

// Model converts the request to a model.
func (r *News) Model() model.News {
	return model.News{
		Title: r.Title, Slug: r.Slug, Excerpt: r.Excerpt, BodyHTML: r.BodyHTML, ThumbnailFileID: r.ThumbnailFileID,
		PublishedAt: r.PublishedAt, Status: r.Status, SEOTitle: r.SEOTitle, MetaDescription: r.MetaDescription,
	}
}
