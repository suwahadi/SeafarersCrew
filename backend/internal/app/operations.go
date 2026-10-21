package app

import (
	"context"
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/job"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/pgdump"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

func (a *App) wireOperations() {
	a.Backups = service.NewBackupService(
		repository.NewBackupRepository(a.DB),
		pgdump.New(""),
		a.Storage,
		a.Settings,
		a.Jobs,
		a.CPU,
		a.Config.DatabaseURL,
		"",
		time.Now,
	)
	a.Jobs.Register(service.BackupJob, a.Backups.Handle)
	a.Maintenance = service.NewMaintenanceService(
		repository.NewMaintenanceRepository(a.DB),
		a.Quotations(),
		a.Dashboard,
		a.Publisher,
		repository.NewCacheRepository(a.Redis),
		a.Storage,
		a.Settings,
		time.Now,
	)
	a.Scheduler = job.NewScheduler(a.scheduledTasks(), repository.NewLockRepository(a.Redis), a.Logger, time.Now)
}

func (a *App) scheduledTasks() []job.Task {
	return []job.Task{
		{Name: "backup", Every: time.Minute, Run: func(ctx context.Context) error {
			_, err := a.Backups.ScheduleIfDue(ctx)
			return err
		}},
		{Name: "close-vacancies", Every: time.Hour, Run: a.Maintenance.CloseVacancies},
		{Name: "expire-quotations", Every: time.Hour, Run: a.Maintenance.ExpireQuotations},
		{Name: "publish-news", Every: time.Minute, Run: a.Maintenance.PublishScheduledNews},
		{Name: "purge-files", Every: 6 * time.Hour, Slot: func(now time.Time) string { return now.UTC().Format("2006-01") }, Run: a.Maintenance.PurgeFiles},
		{Name: "refresh-dashboard", Every: 5 * time.Minute, Run: a.Maintenance.RefreshDashboard},
	}
}
