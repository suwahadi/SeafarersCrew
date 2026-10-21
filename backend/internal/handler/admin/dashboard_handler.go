package admin

import (
	"context"
	"net/http"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

// DashboardService is what the dashboard controller needs.
type DashboardService interface {
	Summary(ctx context.Context) (model.DashboardSummary, service.DashboardContext, error)
	Charts(ctx context.Context) (model.DashboardCharts, error)
	Expiring(ctx context.Context) ([]model.ExpiringItem, service.DashboardContext, error)
	Reliefs(ctx context.Context) ([]model.UpcomingRelief, service.DashboardContext, error)
}

// DashboardHandler serves the dashboard widgets (PRD 6.1).
type DashboardHandler struct {
	dashboard DashboardService
}

// NewDashboardHandler returns a DashboardHandler.
func NewDashboardHandler(dashboard DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard}
}

// Summary handles GET /dashboard/summary.
func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	s, dc, err := h.dashboard.Summary(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewDashboardSummary(s, dc))
}

// Charts handles GET /dashboard/charts.
func (h *DashboardHandler) Charts(w http.ResponseWriter, r *http.Request) {
	c, err := h.dashboard.Charts(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, c)
}

// Expiring handles GET /dashboard/expiring.
func (h *DashboardHandler) Expiring(w http.ResponseWriter, r *http.Request) {
	items, dc, err := h.dashboard.Expiring(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewExpiringItems(items, dc))
}

// Reliefs handles GET /dashboard/reliefs.
func (h *DashboardHandler) Reliefs(w http.ResponseWriter, r *http.Request) {
	items, dc, err := h.dashboard.Reliefs(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewUpcomingReliefs(items, dc))
}
