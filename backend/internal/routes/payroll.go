package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
)

// Payroll mounts payroll periods, their lines, imports and exports on an authenticated admin router.
func Payroll(r chi.Router, h *admin.PayrollHandler) {
	r.Get("/payroll-periods", h.Index)
	r.Post("/payroll-periods", h.Store)
	r.Route("/payroll-periods/{period}", func(r chi.Router) {
		r.Get("/", h.Show)
		r.Put("/", h.Update)
		r.Delete("/", h.Destroy)
		r.Get("/lines", h.Lines)
		r.Post("/lines", h.StoreLine)
		r.Get("/facets", h.Facets)
		r.Post("/import/validate", h.ValidateImport)
		r.Post("/import/commit", h.CommitImport)
		r.Post("/export", h.Export)
	})
	r.Put("/payroll-lines/{id}", h.UpdateLine)
	r.Delete("/payroll-lines/{id}", h.DestroyLine)
}
