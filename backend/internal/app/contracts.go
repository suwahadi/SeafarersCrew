package app

import (
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/routes"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

func (a *App) contractServices() (*service.ContractService, *service.RotationService) {
	contracts := repository.NewContractRepository(a.DB)
	reliefs := repository.NewReliefPlanRepository(a.DB)
	contractService := service.NewContractService(contracts, repository.NewContractLookupRepository(a.DB), a.Crews, reliefs, a.Settings, time.Now)
	rotation := service.NewRotationService(reliefs, contractService, contracts, a.Crews, a.Crews, a.Settings, time.Now)
	return contractService, rotation
}

func (a *App) contractRoutes() routes.ContractRoutes {
	contracts, rotation := a.contractServices()
	return routes.ContractRoutes{
		Contracts: admin.NewContractHandler(contracts),
		Rotation:  admin.NewRotationHandler(rotation),
	}
}
