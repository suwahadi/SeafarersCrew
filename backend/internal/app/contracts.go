package app

import (
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/routes"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

func (a *App) contractRoutes() routes.ContractRoutes {
	contracts := repository.NewContractRepository(a.DB)
	reliefs := repository.NewReliefPlanRepository(a.DB)
	contractService := service.NewContractService(contracts, repository.NewContractLookupRepository(a.DB), a.Crews, reliefs, a.Settings, time.Now)
	rotation := service.NewRotationService(reliefs, contractService, contracts, a.Crews, a.Crews, a.Settings, time.Now)
	return routes.ContractRoutes{
		Contracts: admin.NewContractHandler(contractService),
		Rotation:  admin.NewRotationHandler(rotation),
	}
}
