package app

import (
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/gotenberg"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/pdf"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/routes"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

// Quotations returns the quotation service, also used by the scheduled expiry job.
func (a *App) Quotations() *service.QuotationService {
	return service.NewQuotationService(repository.NewQuotationRepository(a.DB))
}

func (a *App) wireDocuments() {
	a.templates = service.NewDocumentTemplateService(repository.NewDocumentTemplateRepository(a.DB))
	a.Documents = service.NewDocumentService(service.DocumentDeps{
		Templates:  a.templates,
		Sources:    repository.NewDocumentSourceRepository(a.DB),
		Crews:      a.Crews,
		Quotations: a.Quotations(),
		Settings:   a.Settings,
		Files:      service.NewStoredFileReader(repository.NewFileRepository(a.DB), a.Storage),
		Renderer:   gotenberg.NewClient(a.Config.GotenbergURL, 90*time.Second),
		Compressor: pdf.NewProcessor("gs", 2*time.Minute),
		Storage:    a.Files,
		History:    repository.NewGeneratedDocumentRepository(a.DB),
		Pool:       a.CPU,
		Now:        time.Now,
	})
}

func (a *App) documentHandlers() routes.DocumentHandlers {
	return routes.DocumentHandlers{
		Documents:  admin.NewDocumentHandler(a.templates, a.Documents),
		Quotations: admin.NewQuotationHandler(a.Quotations()),
	}
}
