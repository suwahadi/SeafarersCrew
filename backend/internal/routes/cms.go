package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
)

// CMSRoutes carries the website CMS controllers.
type CMSRoutes struct {
	Pages   *admin.PageHandler
	News    *admin.NewsHandler
	Menus   *admin.MenuHandler
	Gallery *admin.GalleryHandler
	Masters map[string]admin.MasterRoutes
}

// CMS mounts the website CMS on an authenticated admin router.
func CMS(r chi.Router, h CMSRoutes) {
	r.Get("/pages", h.Pages.Index)
	r.Post("/pages", h.Pages.Store)
	r.Patch("/pages/reorder", h.Pages.Reorder)
	r.Get("/pages/{id}", h.Pages.Show)
	r.Put("/pages/{id}", h.Pages.Update)
	r.Delete("/pages/{id}", h.Pages.Destroy)

	r.Get("/news", h.News.Index)
	r.Post("/news", h.News.Store)
	r.Get("/news/{id}", h.News.Show)
	r.Put("/news/{id}", h.News.Update)
	r.Delete("/news/{id}", h.News.Destroy)

	r.Get("/menus", h.Menus.Index)
	r.Put("/menus/{menu}", h.Menus.Replace)

	r.Post("/gallery/bulk", h.Gallery.Bulk)
	mountMaster(r, "/gallery", h.Gallery)
	for segment, handler := range h.Masters {
		mountMaster(r, "/"+segment, handler)
	}
}
