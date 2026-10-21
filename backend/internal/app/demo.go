package app

import (
	"github.com/suwahadi/seafarerscrew/backend/database/seeders"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

func (a *App) demoDeps() seeders.DemoDeps {
	masters := newMasterData(a.DB, a.Redis)
	cms := a.cmsServices()
	career := a.careerServices()
	contracts, rotation := a.contractServices()
	return seeders.DemoDeps{
		Pool:       a.DB,
		Logger:     a.Logger,
		Site:       a.Publisher,
		Files:      a.Files,
		Companies:  masters.services.Companies,
		Vessels:    masters.services.Vessels,
		Crews:      a.Crews,
		Contracts:  contracts,
		Rotation:   rotation,
		Quotations: a.Quotations(),
		CMS:        cms.cms,
		News:       cms.news,
		Pages:      cms.pages,
		Vacancies:  career.vacancies,
		Applicants: repository.NewApplicantRepository(a.DB),
		Contacts:   repository.NewContactRepository(a.DB),
	}
}
