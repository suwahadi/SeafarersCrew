// Package routes registers the admin and public API routes.
package routes

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
	"github.com/suwahadi/seafarerscrew/backend/internal/middleware"
)

// AdminPrefix is the base path of the admin API.
const AdminPrefix = "/api/admin/v1"

// AdminDeps carries the controllers and middleware of the admin API.
type AdminDeps struct {
	Origin        string
	Authenticator middleware.Authenticator
	Counter       middleware.HitCounter
	Auth          *admin.AuthHandler
	Settings      *admin.SettingsHandler
	Jobs          *admin.JobHandler
	Crews         *admin.CrewHandler
	Masters       map[string]admin.MasterRoutes
	MasterData    *admin.MasterDataHandler
	Files         *admin.FileHandler
}

// Admin mounts the admin API.
func Admin(r chi.Router, d AdminDeps) {
	r.Route(AdminPrefix, func(r chi.Router) {
		r.Use(middleware.CORS(d.Origin))
		r.Use(middleware.RateLimit(d.Counter, "admin-api", 1200, time.Minute))

		r.With(middleware.RequireJSON).Post("/auth/login", d.Auth.Login)
		r.With(middleware.RequireJSON).Post("/auth/recovery", d.Auth.Recover)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireSession(d.Authenticator))
			r.Use(middleware.CSRF)
			r.Use(middleware.RequirePasswordCurrent(
				http.MethodGet+" "+AdminPrefix+"/me",
				http.MethodPut+" "+AdminPrefix+"/me/password",
				http.MethodPost+" "+AdminPrefix+"/auth/logout",
			))

			r.Get("/me", d.Auth.Me)
			r.Put("/me", d.Auth.UpdateProfile)
			r.Put("/me/password", d.Auth.ChangePassword)
			r.Post("/auth/logout", d.Auth.Logout)
			r.Post("/recovery-codes/regenerate", d.Auth.RegenerateRecoveryCodes)

			r.Get("/settings", d.Settings.Schema)
			r.Get("/settings/{group}", d.Settings.Show)
			r.Put("/settings/{group}", d.Settings.Update)

			r.Get("/options", d.MasterData.Options)
			r.Get("/compliance-matrix", d.MasterData.ComplianceMatrix)
			r.Put("/compliance-matrix", d.MasterData.ReplaceComplianceMatrix)
			for segment, h := range d.Masters {
				mountMaster(r, "/"+segment, h)
			}

			r.Get("/jobs/{id}", d.Jobs.Show)

			Crew(r, d.Crews)
			r.Post("/files", d.Files.Upload)
			r.Get("/files/{id}/url", d.Files.URL)
			r.Delete("/files/{id}", d.Files.Delete)
		})
	})
}

func mountMaster(r chi.Router, path string, h admin.MasterRoutes) {
	r.Get(path, h.List)
	r.Post(path, h.Create)
	r.Patch(path+"/reorder", h.Reorder)
	r.Get(path+"/{id}", h.Show)
	r.Put(path+"/{id}", h.Update)
	r.Delete(path+"/{id}", h.Delete)
}
