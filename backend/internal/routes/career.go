package routes

import (
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
	"github.com/suwahadi/seafarerscrew/backend/internal/handler/public"
	"github.com/suwahadi/seafarerscrew/backend/internal/middleware"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// Public form limits from PRD 8.4.
const (
	FormLimit  = 5
	FormWindow = 10 * time.Minute
)

// CareerRoutes carries the vacancy, applicant and contact inbox controllers.
type CareerRoutes struct {
	Vacancies  *admin.VacancyHandler
	Applicants *admin.ApplicantHandler
	Contact    *admin.ContactHandler
}

// Career mounts vacancies, applicants and the contact inbox on an authenticated admin router.
func Career(r chi.Router, h CareerRoutes) {
	r.Get("/vacancies", h.Vacancies.Index)
	r.Post("/vacancies", h.Vacancies.Store)
	r.Route("/vacancies/{vacancy}", func(r chi.Router) {
		r.Get("/", h.Vacancies.Show)
		r.Put("/", h.Vacancies.Update)
		r.Delete("/", h.Vacancies.Destroy)
		r.Patch("/status", h.Vacancies.UpdateStatus)
	})

	r.Get("/applicants", h.Applicants.Index)
	r.Get("/applicants/stage-counts", h.Applicants.StageCounts)
	r.Route("/applicants/{applicant}", func(r chi.Router) {
		r.Get("/", h.Applicants.Show)
		r.Delete("/", h.Applicants.Destroy)
		r.Patch("/stage", h.Applicants.UpdateStage)
	})

	r.Get("/contact-messages", h.Contact.Index)
	r.Get("/contact-messages/unread-count", h.Contact.UnreadCount)
	r.Route("/contact-messages/{id}", func(r chi.Router) {
		r.Get("/", h.Contact.Show)
		r.Patch("/", h.Contact.Update)
	})
}

// PublicCareerRoutes carries the public career and form controllers.
type PublicCareerRoutes struct {
	Vacancies    *public.VacancyHandler
	Applications *public.ApplicationHandler
	Contact      *public.ContactHandler
}

// PublicCareer mounts the career pages and public forms on the rate-limited public router.
func PublicCareer(r chi.Router, d PublicDeps) {
	h := d.Career
	vacancies := cached(d, model.TagVacancies)
	r.With(vacancies).Get("/vacancies", h.Vacancies.Index)
	r.With(vacancies).Get("/vacancies/{slug}", h.Vacancies.Show)
	r.With(vacancies).Get("/vacancy-filters", h.Vacancies.Filters)
	r.With(middleware.RateLimit(d.Counter, "applications", FormLimit, FormWindow)).Post("/applications", h.Applications.Store)
	r.With(
		middleware.RateLimit(d.Counter, "contact", FormLimit, FormWindow),
		middleware.RequireJSON,
	).Post("/contact", h.Contact.Store)
}
