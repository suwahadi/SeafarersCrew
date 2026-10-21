package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

type countingDashboard struct {
	summaryCalls int
	today        time.Time
	thresholds   [3]int
}

func (c *countingDashboard) Summary(_ context.Context, today time.Time, t [3]int) (model.DashboardSummary, error) {
	c.summaryCalls++
	c.today, c.thresholds = today, t
	return model.DashboardSummary{TotalCrew: c.summaryCalls}, nil
}

func (c *countingDashboard) Charts(context.Context, time.Time) (model.DashboardCharts, error) {
	return model.DashboardCharts{}, nil
}

func (c *countingDashboard) Expiring(context.Context, time.Time, int, int) ([]model.ExpiringItem, error) {
	return []model.ExpiringItem{{Number: "X1", ExpiredDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}}, nil
}

func (c *countingDashboard) Reliefs(context.Context, time.Time, int, int) ([]model.UpcomingRelief, error) {
	return nil, nil
}

func newDashboard() (*DashboardService, *countingDashboard) {
	store := &countingDashboard{}
	settings := staticSettings{
		"general": {"timezone": "Asia/Jakarta"},
		"alerts":  {"expiry_threshold_1": 14, "expiry_threshold_2": 45, "expiry_threshold_3": 120},
	}
	now := func() time.Time { return time.Date(2026, 10, 20, 20, 0, 0, 0, time.UTC) }
	return NewDashboardService(store, newMemCache(), settings, now), store
}

func TestDashboardUsesTodayInWIBAndConfiguredThresholds(t *testing.T) {
	svc, store := newDashboard()
	_, dc, err := svc.Summary(context.Background())
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC), store.today)
	assert.Equal(t, [3]int{14, 45, 120}, store.thresholds)
	assert.Equal(t, store.today, dc.Today)
}

func TestDashboardCachesUntilInvalidated(t *testing.T) {
	svc, store := newDashboard()
	ctx := context.Background()
	first, _, err := svc.Summary(ctx)
	require.NoError(t, err)
	second, _, err := svc.Summary(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, store.summaryCalls)
	assert.Equal(t, first, second)

	require.NoError(t, svc.Invalidate(ctx))
	third, _, err := svc.Summary(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, store.summaryCalls)
	assert.Equal(t, 2, third.TotalCrew)
}

func TestDashboardCacheKeepsDates(t *testing.T) {
	svc, _ := newDashboard()
	ctx := context.Background()
	_, _, err := svc.Expiring(ctx)
	require.NoError(t, err)
	cached, _, err := svc.Expiring(ctx)
	require.NoError(t, err)
	require.Len(t, cached, 1)
	assert.Equal(t, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), cached[0].ExpiredDate.UTC())
}
