package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

func mountPublicContent(r chi.Router, d PublicDeps) {
	r.With(cached(d, model.TagSite)).Get("/site", d.Content.Site)
	r.With(cached(d, model.TagHome, model.TagNews, model.TagGallery, model.TagVacancies)).Get("/home", d.Content.Home)
	r.With(cached(d, model.TagPages, model.TagGallery)).Get("/pages/{slug}", d.Content.Page)
	r.With(cached(d, model.TagNews)).Get("/news", d.Content.NewsIndex)
	r.With(cached(d, model.TagNews)).Get("/news/{slug}", d.Content.NewsShow)
	r.With(cached(d, model.TagGallery)).Get("/gallery", d.Content.Gallery)
	r.With(cached(d, model.TagSitemap, model.TagPages, model.TagNews, model.TagVacancies)).Get("/sitemap-data", d.Content.Sitemap)
}
