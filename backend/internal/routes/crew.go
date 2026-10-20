package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

// Crew mounts crew records and their tabs on an authenticated admin router.
func Crew(r chi.Router, h *admin.CrewHandler, t *admin.CrewTransferHandler) {
	r.Get("/crews", h.Index)
	r.Post("/crews", h.Store)
	r.Post("/crews/export", t.Export)
	r.Get("/crews/import-template", t.Template)
	r.Post("/crews/import/validate", t.Validate)
	r.Post("/crews/import/commit", t.Commit)
	r.Route("/crews/{crew}", func(r chi.Router) {
		r.Get("/", h.Show)
		r.Put("/", h.Update)
		r.Delete("/", h.Destroy)
		r.Patch("/status", h.Status)
		r.Get("/compliance", h.Compliance)
		r.Get("/sea-service", h.SeaService)

		r.Get("/bank-accounts", h.BankAccounts)
		r.Post("/bank-accounts", h.SaveBankAccount)
		r.Put("/bank-accounts/{id}", h.SaveBankAccount)
		r.Delete("/bank-accounts/{id}", h.DeleteBankAccount)

		r.Get("/next-of-kins", h.NextOfKins)
		r.Post("/next-of-kins", h.SaveNextOfKin)
		r.Put("/next-of-kins/{id}", h.SaveNextOfKin)
		r.Delete("/next-of-kins/{id}", h.DeleteNextOfKin)

		listDocs, saveDoc, deleteDoc := h.Documents(repository.TravelDocuments)
		r.Get("/documents", listDocs)
		r.Post("/documents", saveDoc)
		r.Put("/documents/{id}", saveDoc)
		r.Delete("/documents/{id}", deleteDoc)

		listCerts, saveCert, deleteCert := h.Documents(repository.Certificates)
		r.Get("/certificates", listCerts)
		r.Post("/certificates", saveCert)
		r.Put("/certificates/{id}", saveCert)
		r.Delete("/certificates/{id}", deleteCert)

		r.Get("/experiences", h.SeaService)
		r.Post("/experiences", h.SaveExperience)
		r.Put("/experiences/{id}", h.SaveExperience)
		r.Delete("/experiences/{id}", h.DeleteExperience)
	})
}
