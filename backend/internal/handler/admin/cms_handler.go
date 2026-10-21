package admin

import (
	"context"
	"net/http"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

// CMSMasters builds the handlers of the simple CMS lists keyed by URL segment; gallery is returned separately for its bulk route.
func CMSMasters(c service.CMS, urls resource.MediaURLer) (map[string]MasterRoutes, *GalleryHandler) {
	masters := map[string]MasterRoutes{
		"hero-slides":  NewMasterHandler[model.HeroSlide, request.HeroSlide](c.HeroSlides, present(resource.NewHeroSlide(urls)), nil),
		"testimonials": NewMasterHandler[model.Testimonial, request.Testimonial](c.Testimonials, present(resource.NewTestimonial(urls)), nil),
		"client-logos": NewMasterHandler[model.ClientLogo, request.ClientLogo](c.ClientLogos, present(resource.NewClientLogo(urls)), nil),
		"stats":        NewMasterHandler[model.StatsItem, request.StatsItem](c.Stats, present(resource.NewStatsItem), nil),
		"languages":    NewMasterHandler[model.Language, request.Language](c.Languages, present(resource.NewLanguage(urls)), nil),
	}
	return masters, NewGalleryHandler(c.Gallery, urls)
}

// GalleryService is the gallery contract: the shared CRUD plus bulk creation.
type GalleryService interface {
	MasterCRUD[model.GalleryImage]
	BulkCreate(ctx context.Context, items []model.GalleryImage) ([]model.GalleryImage, error)
}

// GalleryHandler serves the gallery CRUD and bulk creation.
type GalleryHandler struct {
	*MasterHandler[model.GalleryImage, request.GalleryImage, *request.GalleryImage]
	svc     GalleryService
	present func(model.GalleryImage) resource.GalleryImage
}

// NewGalleryHandler returns a GalleryHandler.
func NewGalleryHandler(svc GalleryService, urls resource.MediaURLer) *GalleryHandler {
	mapper := resource.NewGalleryImage(urls)
	return &GalleryHandler{
		MasterHandler: NewMasterHandler[model.GalleryImage, request.GalleryImage](svc, present(mapper),
			map[string]Filter{"show_on_home": {Kind: FilterBool}}),
		svc:     svc,
		present: mapper,
	}
}

// Bulk handles POST /gallery/bulk.
func (h *GalleryHandler) Bulk(w http.ResponseWriter, r *http.Request) {
	var in request.GalleryBulk
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	created, err := h.svc.BulkCreate(r.Context(), in.Model())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusCreated, resource.MapAll(created, h.present))
}
