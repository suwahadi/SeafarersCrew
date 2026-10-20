package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
)

// Email mounts email dispatch, logs, templates and Settings connection tests on an authenticated admin router.
func Email(r chi.Router, h *admin.EmailHandler) {
	r.Post("/emails/send", h.Send)
	r.Get("/email-logs", h.Logs)
	r.Post("/email-logs/{id}/resend", h.Resend)
	r.Get("/email-templates", h.Templates)
	r.Get("/email-templates/{code}", h.Template)
	r.Put("/email-templates/{code}", h.UpdateTemplate)
	r.Post("/email-templates/{code}/preview", h.Preview)
	r.Post("/settings/{group}/test", h.TestConnection)
}
