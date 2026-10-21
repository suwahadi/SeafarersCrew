package app

import (
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
	"github.com/suwahadi/seafarerscrew/backend/internal/handler/public"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/turnstile"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/routes"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

type careerServices struct {
	vacancies    *service.VacancyService
	applicants   *service.ApplicantService
	applications *service.ApplicationService
	contact      *service.ContactService
}

func (a *App) careerServices() careerServices {
	vacancyRepo := repository.NewVacancyRepository(a.DB)
	applicantRepo := repository.NewApplicantRepository(a.DB)
	guard := service.NewFormGuard(turnstile.New(turnstile.DefaultURL, integrationTimeout), a.Settings, a.Config.IsLocal())
	return careerServices{
		vacancies:  service.NewVacancyService(vacancyRepo, a.Settings, a.Publisher, time.Now),
		applicants: service.NewApplicantService(applicantRepo, a.Settings, time.Now),
		applications: service.NewApplicationService(applicantRepo, vacancyRepo, a.Files, guard, a.Email, a.Dashboard,
			a.Settings, a.Config.AdminURL, time.Now),
		contact: service.NewContactService(repository.NewContactRepository(a.DB), guard, a.Email, a.Dashboard, a.Config.AdminURL),
	}
}

func (a *App) careerRoutes(s careerServices) routes.CareerRoutes {
	return routes.CareerRoutes{
		Vacancies:  admin.NewVacancyHandler(s.vacancies),
		Applicants: admin.NewApplicantHandler(s.applicants),
		Contact:    admin.NewContactHandler(s.contact),
	}
}

func (a *App) publicCareerRoutes(s careerServices) routes.PublicCareerRoutes {
	return routes.PublicCareerRoutes{
		Vacancies:    public.NewVacancyHandler(s.vacancies),
		Applications: public.NewApplicationHandler(s.applications),
		Contact:      public.NewContactHandler(s.contact),
	}
}
