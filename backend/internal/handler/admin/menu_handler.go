package admin

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
)

// MenuService is what the menus controller needs.
type MenuService interface {
	Get(ctx context.Context) (model.Menus, error)
	Replace(ctx context.Context, menu string, items []model.MenuItem) (model.Menus, error)
}

// MenuHandler serves the header and footer menus.
type MenuHandler struct {
	svc MenuService
}

// NewMenuHandler returns a MenuHandler.
func NewMenuHandler(svc MenuService) *MenuHandler {
	return &MenuHandler{svc: svc}
}

// Index handles GET /menus.
func (h *MenuHandler) Index(w http.ResponseWriter, r *http.Request) {
	menus, err := h.svc.Get(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewMenus(menus))
}

// Replace handles PUT /menus/{menu}.
func (h *MenuHandler) Replace(w http.ResponseWriter, r *http.Request) {
	var in request.Menu
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	menus, err := h.svc.Replace(r.Context(), chi.URLParam(r, "menu"), in.Model())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewMenus(menus))
}
