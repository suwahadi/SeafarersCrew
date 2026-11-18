package app

import (
	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

func (a *App) wirePayroll() {
	store := repository.NewPayrollRepository(a.DB)
	imports := service.NewPayrollImportService(store, repository.NewCacheRepository(a.Redis), a.Jobs)
	exports := service.NewPayrollExportService(store, a.Jobs, a.Files,
		service.NewStoredFileReader(repository.NewFileRepository(a.DB), a.Storage), a.Settings)
	a.Jobs.Register(service.PayrollImportJob, imports.Handle)
	a.Jobs.Register(service.PayrollExportJob, exports.Handle)
	a.payroll = admin.NewPayrollHandler(service.NewPayrollService(store), imports, exports, a.Files)
}
