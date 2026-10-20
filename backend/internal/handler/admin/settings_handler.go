package admin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

// SettingsService is what the settings controller needs.
type SettingsService interface {
	AdminView(ctx context.Context, group string) (map[string]any, error)
	Update(ctx context.Context, group string, input map[string]json.RawMessage) (map[string]any, error)
}

// SettingsHandler serves the Settings panel.
type SettingsHandler struct {
	settings SettingsService
}

// NewSettingsHandler returns a SettingsHandler.
func NewSettingsHandler(settings SettingsService) *SettingsHandler {
	return &SettingsHandler{settings: settings}
}

// Schema handles GET /settings and describes every group and field.
func (h *SettingsHandler) Schema(w http.ResponseWriter, _ *http.Request) {
	resource.JSON(w, http.StatusOK, service.SettingGroups)
}

// Show handles GET /settings/{group}.
func (h *SettingsHandler) Show(w http.ResponseWriter, r *http.Request) {
	values, err := h.settings.AdminView(r.Context(), chi.URLParam(r, "group"))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, values)
}

// Update handles PUT /settings/{group}.
func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var in map[string]json.RawMessage
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	values, err := h.settings.Update(r.Context(), chi.URLParam(r, "group"), in)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, values)
}
