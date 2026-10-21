package public

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
)

// ContentService assembles the public site data.
type ContentService interface {
	Site(ctx context.Context) (model.SiteData, error)
	Home(ctx context.Context) (model.HomeData, error)
	Page(ctx context.Context, slug string) (model.PageData, error)
	NewsList(ctx context.Context, page, perPage int) (model.NewsPageData, error)
	NewsDetail(ctx context.Context, slug string) (model.NewsDetailData, error)
	Gallery(ctx context.Context) ([]model.GalleryImage, error)
	Sitemap(ctx context.Context) (model.SitemapData, error)
}

// ContentHandler serves site, home, pages, news, gallery and sitemap data.
type ContentHandler struct {
	svc  ContentService
	urls resource.MediaURLer
}

// NewContentHandler builds a ContentHandler.
func NewContentHandler(svc ContentService, urls resource.MediaURLer) *ContentHandler {
	return &ContentHandler{svc: svc, urls: urls}
}

// Site handles GET /site.
func (h *ContentHandler) Site(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.Site(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewPublicSite(data, h.urls))
}

// Home handles GET /home.
func (h *ContentHandler) Home(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.Home(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewPublicHome(data, h.urls))
}

// Page handles GET /pages/{slug}.
func (h *ContentHandler) Page(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.Page(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewPublicPage(data, h.urls))
}

// NewsIndex handles GET /news.
func (h *ContentHandler) NewsIndex(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	data, err := h.svc.NewsList(r.Context(), page, perPage)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.List(w, resource.MapAll(data.Articles, resource.NewPublicNewsCard(h.urls)), resource.Meta{Page: data.Page, PerPage: data.PerPage, Total: data.Total})
}

// NewsShow handles GET /news/{slug}.
func (h *ContentHandler) NewsShow(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.NewsDetail(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewPublicNewsArticle(data, h.urls))
}

// Gallery handles GET /gallery.
func (h *ContentHandler) Gallery(w http.ResponseWriter, r *http.Request) {
	images, err := h.svc.Gallery(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.MapAll(images, resource.NewPublicGalleryImage(h.urls)))
}

// Sitemap handles GET /sitemap-data.
func (h *ContentHandler) Sitemap(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.Sitemap(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewPublicSitemap(data))
}
