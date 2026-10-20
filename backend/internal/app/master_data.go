package app

import (
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

type masterDataDeps struct {
	services   service.MasterData
	options    *service.MasterOptionsService
	compliance *service.ComplianceMatrixService
}

func newMasterData(db *pgxpool.Pool, rdb *goredis.Client) masterDataDeps {
	companies := repository.NewCompanyRepository(db)
	vessels := repository.NewVesselRepository(db)
	ranks := repository.NewRankRepository(db)
	vesselTypes := repository.NewVesselTypeRepository(db)
	currencies := repository.NewCurrencyRepository(db)
	crewStatuses := repository.NewCrewStatusRepository(db)
	allowanceTypes := repository.NewAllowanceTypeRepository(db)
	documentTypes := repository.NewDocumentTypeRepository(db)
	certificateTypes := repository.NewCertificateTypeRepository(db)

	options := service.NewMasterOptionsService(service.MasterListers{
		Companies: companies, Vessels: vessels, Ranks: ranks, VesselTypes: vesselTypes, Currencies: currencies,
		CrewStatuses: crewStatuses, AllowanceTypes: allowanceTypes, DocumentTypes: documentTypes, CertificateTypes: certificateTypes,
	}, repository.NewCacheRepository(rdb))

	services := service.NewMasterData(service.MasterStores{
		Companies: companies, Vessels: vessels, Ranks: ranks, VesselTypes: vesselTypes, Currencies: currencies,
		CrewStatuses: crewStatuses, AllowanceTypes: allowanceTypes, DocumentTypes: documentTypes, CertificateTypes: certificateTypes,
	}, repository.NewLookupRepository(db), options.Invalidate)

	return masterDataDeps{
		services:   services,
		options:    options,
		compliance: service.NewComplianceMatrixService(repository.NewComplianceMatrixRepository(db), ranks, documentTypes, certificateTypes),
	}
}
