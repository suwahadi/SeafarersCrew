package service

import (
	"context"
	"log/slog"
	"slices"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

var publicSettingGroups = []string{"general", "company", "contact", "branding", "seo", "website", "security"}

// SetOnChanged installs the hook that runs after a settings group is saved; call it once while wiring.
func (s *SettingsService) SetOnChanged(fn SettingsChanged) {
	s.onChanged = fn
}

// PublishSettingChanges returns a settings hook that refreshes the public site when a group it displays is saved.
func PublishSettingChanges(site SiteChanger, logger *slog.Logger) SettingsChanged {
	return func(ctx context.Context, group string) {
		if !slices.Contains(publicSettingGroups, group) {
			return
		}
		err := site.Changed(ctx, model.TagSite, model.TagHome, model.TagPages, model.TagNews, model.TagGallery, model.TagVacancies, model.TagSitemap)
		if err != nil {
			logger.WarnContext(ctx, "public site not refreshed after settings change", "group", group, "error", err)
		}
	}
}
