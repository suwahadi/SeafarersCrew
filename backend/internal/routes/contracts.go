package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
)

// ContractRoutes carries the contract and rotation controllers.
type ContractRoutes struct {
	Contracts *admin.ContractHandler
	Rotation  *admin.RotationHandler
}

// Contracts mounts contracts, the current assignment tab and rotation planning on an authenticated admin router.
func Contracts(r chi.Router, h ContractRoutes) {
	r.Get("/contracts", h.Contracts.Index)
	r.Post("/contracts", h.Contracts.Store)
	r.Route("/contracts/{contract}", func(r chi.Router) {
		r.Get("/", h.Contracts.Show)
		r.Put("/", h.Contracts.Update)
		r.Delete("/", h.Contracts.Destroy)
		r.Post("/activate", h.Contracts.Activate)
		r.Post("/complete", h.Contracts.Complete)
		r.Post("/terminate", h.Contracts.Terminate)
	})
	r.Get("/crews/{crew}/current-assignment", h.Contracts.CurrentAssignment)

	r.Get("/rotation/timeline", h.Rotation.Timeline)
	r.Get("/relief-plans", h.Rotation.Index)
	r.Post("/relief-plans", h.Rotation.Store)
	r.Get("/relief-plans/suggestions", h.Rotation.Suggestions)
	r.Get("/relief-plans/{id}", h.Rotation.Show)
	r.Put("/relief-plans/{id}", h.Rotation.Update)
	r.Delete("/relief-plans/{id}", h.Rotation.Destroy)
	r.Post("/relief-plans/{id}/complete", h.Rotation.Complete)
}
