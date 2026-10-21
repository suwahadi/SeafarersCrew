package service

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

func TestSavingPublicSettingGroupsRefreshesTheWholePublicSite(t *testing.T) {
	for _, group := range []string{"general", "company", "contact", "branding", "seo", "website", "security"} {
		t.Run(group, func(t *testing.T) {
			site := &siteRecorder{}
			PublishSettingChanges(site, slog.New(slog.NewTextHandler(io.Discard, nil)))(context.Background(), group)
			assert.ElementsMatch(t, []string{
				model.TagSite, model.TagHome, model.TagPages, model.TagNews, model.TagGallery, model.TagVacancies, model.TagSitemap,
			}, site.last())
		})
	}
}

func TestSavingPrivateSettingGroupsLeavesThePublicSiteAlone(t *testing.T) {
	for _, group := range []string{"email", "storage", "uploads", "backup", "alerts"} {
		site := &siteRecorder{}
		PublishSettingChanges(site, slog.New(slog.NewTextHandler(io.Discard, nil)))(context.Background(), group)
		assert.Zero(t, site.count(), group)
	}
}

func TestSettingsServiceRunsTheInstalledHookAfterASave(t *testing.T) {
	svc, _, _ := newSettingsFixture(t)
	var groups []string
	svc.SetOnChanged(func(_ context.Context, group string) { groups = append(groups, group) })

	_, err := svc.Update(context.Background(), "branding", map[string]json.RawMessage{"primary_color": raw("#112233")})
	require.NoError(t, err)
	assert.Equal(t, []string{"branding"}, groups)
}
