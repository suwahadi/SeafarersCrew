package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/logctx"
)

const (
	purgeAfterDays   = 90
	purgeBatchSize   = 200
	purgeMaxBatches  = 50
	newsCursorKey    = "sched:news-cursor"
	newsCursorExpiry = 30 * 24 * time.Hour
)

// MaintenanceStore holds the statements of the scheduled jobs.
type MaintenanceStore interface {
	CloseExpiredVacancies(ctx context.Context, today time.Time) (int64, error)
	CountNewlyLiveNews(ctx context.Context, from, to time.Time) (int, error)
	PurgeCandidates(ctx context.Context, cutoff time.Time, limit int) ([]model.File, error)
	MarkPurged(ctx context.Context, ids []uuid.UUID, at time.Time) error
}

// QuotationExpirer expires overdue quotations.
type QuotationExpirer interface {
	ExpireQuotations(ctx context.Context, today time.Time) (int64, error)
}

// Invalidator drops cached dashboard figures.
type Invalidator interface {
	Invalidate(ctx context.Context) error
}

// SiteChangePublisher announces public content changes.
type SiteChangePublisher interface {
	Changed(ctx context.Context, tags ...string) error
}

// CursorStore remembers a small value between runs.
type CursorStore interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// MaintenanceService runs the periodic housekeeping of PRD 15.3.
type MaintenanceService struct {
	store      MaintenanceStore
	quotations QuotationExpirer
	dashboard  Invalidator
	site       SiteChangePublisher
	cursor     CursorStore
	storage    BackupStorage
	settings   SettingsReader
	now        func() time.Time
}

// NewMaintenanceService builds a MaintenanceService.
func NewMaintenanceService(store MaintenanceStore, quotations QuotationExpirer, dashboard Invalidator, site SiteChangePublisher, cursor CursorStore, storage BackupStorage, settings SettingsReader, now func() time.Time) *MaintenanceService {
	return &MaintenanceService{store: store, quotations: quotations, dashboard: dashboard, site: site, cursor: cursor, storage: storage, settings: settings, now: now}
}

func (s *MaintenanceService) today(ctx context.Context) (time.Time, error) {
	loc, _, err := LoadDisplaySettings(ctx, s.settings)
	if err != nil {
		return time.Time{}, err
	}
	return datetime.Today(s.now(), loc), nil
}

// CloseVacancies closes open vacancies past their closing date and refreshes the site when any closed.
func (s *MaintenanceService) CloseVacancies(ctx context.Context) error {
	today, err := s.today(ctx)
	if err != nil {
		return err
	}
	n, err := s.store.CloseExpiredVacancies(ctx, today)
	if err != nil || n == 0 {
		return err
	}
	logctx.From(ctx).InfoContext(ctx, "vacancies closed", "count", n)
	if err := s.site.Changed(ctx, model.TagVacancies, model.TagHome, model.TagSitemap); err != nil {
		return err
	}
	return s.dashboard.Invalidate(ctx)
}

// ExpireQuotations expires quotations past their valid-until date.
func (s *MaintenanceService) ExpireQuotations(ctx context.Context) error {
	today, err := s.today(ctx)
	if err != nil {
		return err
	}
	n, err := s.quotations.ExpireQuotations(ctx, today)
	if err != nil || n == 0 {
		return err
	}
	logctx.From(ctx).InfoContext(ctx, "quotations expired", "count", n)
	return s.dashboard.Invalidate(ctx)
}

// PublishScheduledNews refreshes the site when a scheduled article has reached its publish time since the last run.
func (s *MaintenanceService) PublishScheduledNews(ctx context.Context) error {
	now := s.now().UTC()
	from := now.Add(-time.Minute)
	if raw, ok, err := s.cursor.Get(ctx, newsCursorKey); err != nil {
		return err
	} else if ok {
		if t, perr := time.Parse(time.RFC3339Nano, string(raw)); perr == nil {
			from = t
		}
	}
	n, err := s.store.CountNewlyLiveNews(ctx, from, now)
	if err != nil {
		return err
	}
	if n > 0 {
		if err := s.site.Changed(ctx, model.TagNews, model.TagHome, model.TagSitemap); err != nil {
			return err
		}
	}
	return s.cursor.Set(ctx, newsCursorKey, []byte(now.Format(time.RFC3339Nano)), newsCursorExpiry)
}

// PurgeFiles deletes the stored files of records removed more than 90 days ago.
func (s *MaintenanceService) PurgeFiles(ctx context.Context) error {
	cutoff := s.now().UTC().AddDate(0, 0, -purgeAfterDays)
	total := 0
	for range purgeMaxBatches {
		files, err := s.store.PurgeCandidates(ctx, cutoff, purgeBatchSize)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			break
		}
		ids := make([]uuid.UUID, 0, len(files))
		var keys []string
		for _, f := range files {
			ids = append(ids, f.ID)
			keys = append(keys, f.StorageKey)
			for _, v := range f.Variants {
				keys = append(keys, v.StorageKey)
			}
		}
		if err := s.storage.Delete(ctx, keys...); err != nil {
			return fmt.Errorf("delete purged objects: %w", err)
		}
		if err := s.store.MarkPurged(ctx, ids, s.now().UTC()); err != nil {
			return err
		}
		total += len(files)
		if len(files) < purgeBatchSize {
			break
		}
	}
	if total > 0 {
		logctx.From(ctx).InfoContext(ctx, "files purged", "count", total)
	}
	return nil
}

// RefreshDashboard drops cached dashboard figures so they are recomputed on the next view.
func (s *MaintenanceService) RefreshDashboard(ctx context.Context) error {
	return s.dashboard.Invalidate(ctx)
}
