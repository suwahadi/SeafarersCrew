package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
)

// DocumentHandlers groups the document builder and quotation controllers.
type DocumentHandlers struct {
	Documents  *admin.DocumentHandler
	Quotations *admin.QuotationHandler
}

// Documents mounts templates, document generation and quotations on an authenticated admin router.
func Documents(r chi.Router, h DocumentHandlers) {
	r.Get("/document-templates", h.Documents.Templates)
	r.Get("/document-templates/variables", h.Documents.Variables)
	r.Post("/document-templates", h.Documents.SaveTemplate)
	r.Get("/document-templates/{id}", h.Documents.Template)
	r.Put("/document-templates/{id}", h.Documents.SaveTemplate)
	r.Delete("/document-templates/{id}", h.Documents.DeleteTemplate)

	r.Post("/documents/preview", h.Documents.Preview)
	r.Post("/documents/generate", h.Documents.Generate)
	r.Get("/generated-documents", h.Documents.History)

	r.Get("/quotations", h.Quotations.Index)
	r.Post("/quotations", h.Quotations.Store)
	r.Get("/quotations/{id}", h.Quotations.Show)
	r.Put("/quotations/{id}", h.Quotations.Update)
	r.Patch("/quotations/{id}/status", h.Quotations.Status)
	r.Delete("/quotations/{id}", h.Quotations.Destroy)
}
