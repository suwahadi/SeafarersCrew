package model

// Cache tags shared by the public API cache and Next.js revalidation.
const (
	TagSite      = "site"
	TagHome      = "home"
	TagPages     = "pages"
	TagNews      = "news"
	TagGallery   = "gallery"
	TagVacancies = "vacancies"
	TagRedirects = "redirects"
	TagSitemap   = "sitemap"
)

// Redirect maps an old public path to its current one.
type Redirect struct {
	FromPath   string `json:"from_path"`
	ToPath     string `json:"to_path"`
	StatusCode int    `json:"status_code"`
}
