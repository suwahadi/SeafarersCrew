package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/token"
)

// Dashboard cache settings from PRD 6.1.
const (
	DashboardCacheTTL  = 5 * time.Minute
	dashboardVersion   = "dashboard:version"
	dashboardListLimit = 50
)

// DashboardStore runs the dashboard aggregate queries.
type DashboardStore interface {
	Summary(ctx context.Context, today time.Time, t [3]int) (model.DashboardSummary, error)
	Charts(ctx context.Context, today time.Time) (model.DashboardCharts, error)
	Expiring(ctx context.Context, today time.Time, days, limit int) ([]model.ExpiringItem, error)
	Reliefs(ctx context.Context, today time.Time, days, limit int) ([]model.UpcomingRelief, error)
}

// DashboardService serves cached dashboard aggregates.
type DashboardService struct {
	store    DashboardStore
	cache    Cache
	settings SettingsReader
	now      func() time.Time
}

// NewDashboardService builds a DashboardService.
func NewDashboardService(store DashboardStore, cache Cache, settings SettingsReader, now func() time.Time) *DashboardService {
	return &DashboardService{store: store, cache: cache, settings: settings, now: now}
}

// DashboardContext is the date and thresholds the dashboard was computed with.
type DashboardContext struct {
	Today      time.Time
	Thresholds Thresholds
}

// Summary returns the KPI cards.
func (s *DashboardService) Summary(ctx context.Context) (model.DashboardSummary, DashboardContext, error) {
	dc, err := s.context(ctx)
	if err != nil {
		return model.DashboardSummary{}, dc, err
	}
	var out model.DashboardSummary
	err = s.cached(ctx, "summary", dc, &out, func() (any, error) { return s.store.Summary(ctx, dc.Today, dc.Thresholds) })
	return out, dc, err
}

// Charts returns the chart series.
func (s *DashboardService) Charts(ctx context.Context) (model.DashboardCharts, error) {
	dc, err := s.context(ctx)
	if err != nil {
		return model.DashboardCharts{}, err
	}
	var out model.DashboardCharts
	err = s.cached(ctx, "charts", dc, &out, func() (any, error) { return s.store.Charts(ctx, dc.Today) })
	return out, err
}

// Expiring returns documents and certificates expiring within the largest threshold, nearest first.
func (s *DashboardService) Expiring(ctx context.Context) ([]model.ExpiringItem, DashboardContext, error) {
	dc, err := s.context(ctx)
	if err != nil {
		return nil, dc, err
	}
	out := []model.ExpiringItem{}
	err = s.cached(ctx, "expiring", dc, &out, func() (any, error) {
		return s.store.Expiring(ctx, dc.Today, dc.Thresholds[2], dashboardListLimit)
	})
	return out, dc, err
}

// Reliefs returns active contracts due for relief in the next 30 days or overdue.
func (s *DashboardService) Reliefs(ctx context.Context) ([]model.UpcomingRelief, DashboardContext, error) {
	dc, err := s.context(ctx)
	if err != nil {
		return nil, dc, err
	}
	out := []model.UpcomingRelief{}
	err = s.cached(ctx, "reliefs", dc, &out, func() (any, error) { return s.store.Reliefs(ctx, dc.Today, 30, dashboardListLimit) })
	return out, dc, err
}

// Invalidate makes every cached dashboard aggregate stale.
func (s *DashboardService) Invalidate(ctx context.Context) error {
	v, err := token.URLSafe(6)
	if err != nil {
		return err
	}
	return s.cache.Set(ctx, dashboardVersion, []byte(v), 30*24*time.Hour)
}

func (s *DashboardService) context(ctx context.Context) (DashboardContext, error) {
	loc, t, err := LoadDisplaySettings(ctx, s.settings)
	if err != nil {
		return DashboardContext{}, err
	}
	return DashboardContext{Today: datetime.Today(s.now(), loc), Thresholds: t}, nil
}

func (s *DashboardService) cached(ctx context.Context, name string, dc DashboardContext, dst any, load func() (any, error)) error {
	version, _, err := s.cache.Get(ctx, dashboardVersion)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("dashboard:%s:%s:%s:%s", version, name, dc.Today.Format(datetime.DateLayout), thresholdKey(dc.Thresholds))
	if raw, ok, err := s.cache.Get(ctx, key); err != nil {
		return err
	} else if ok && json.Unmarshal(raw, dst) == nil {
		return nil
	}
	v, err := load()
	if err != nil {
		return err
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode dashboard %s: %w", name, err)
	}
	if err := s.cache.Set(ctx, key, raw, DashboardCacheTTL); err != nil {
		return err
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("decode dashboard %s: %w", name, err)
	}
	return nil
}

func thresholdKey(t Thresholds) string {
	return strconv.Itoa(t[0]) + "-" + strconv.Itoa(t[1]) + "-" + strconv.Itoa(t[2])
}
