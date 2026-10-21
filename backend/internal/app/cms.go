package app

import (
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
	"github.com/suwahadi/seafarerscrew/backend/internal/handler/public"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/routes"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

func (a *App) cmsRoutes() routes.CMSRoutes {
	lookup := repository.NewCMSLookupRepository(a.DB)
	files := service.NewPublicFileChecker(lookup)
	html := service.NewCMSPolicy()
	cms := service.NewCMS(service.CMSStores{
		HeroSlides:   repository.NewHeroSlideRepository(a.DB),
		Gallery:      repository.NewGalleryImageRepository(a.DB),
		Testimonials: repository.NewTestimonialRepository(a.DB),
		ClientLogos:  repository.NewClientLogoRepository(a.DB),
		Stats:        repository.NewStatsItemRepository(a.DB),
		Languages:    repository.NewLanguageRepository(a.DB),
	}, files, a.Publisher)
	masters, gallery := admin.CMSMasters(cms, a.Media)
	pages := service.NewPageService(repository.NewPageRepository(a.DB), lookup, files, html, a.Publisher, time.Now)
	news := service.NewNewsService(repository.NewNewsRepository(a.DB), files, html, a.Publisher, time.Now)
	menus := service.NewMenuService(repository.NewMenuRepository(a.DB), lookup, a.Publisher)
	return routes.CMSRoutes{
		Pages:   admin.NewPageHandler(pages, a.Media),
		News:    admin.NewNewsHandler(news, a.Media, time.Now),
		Menus:   admin.NewMenuHandler(menus),
		Gallery: gallery,
		Masters: masters,
	}
}

func (a *App) publicContent() *public.ContentHandler {
	lists := service.PublicLists{
		Slides:       repository.NewHeroSlideRepository(a.DB),
		Stats:        repository.NewStatsItemRepository(a.DB),
		Testimonials: repository.NewTestimonialRepository(a.DB),
		Logos:        repository.NewClientLogoRepository(a.DB),
		Gallery:      repository.NewGalleryImageRepository(a.DB),
		Languages:    repository.NewLanguageRepository(a.DB),
	}
	svc := service.NewPublicService(
		a.Settings,
		repository.NewPublicContentRepository(a.DB),
		repository.NewMenuRepository(a.DB),
		repository.NewCMSLookupRepository(a.DB),
		lists,
		time.Now,
	)
	return public.NewContentHandler(svc, a.Media)
}
