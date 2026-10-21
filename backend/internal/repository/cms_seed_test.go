package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/database/seeders"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

func TestReferenceSeedCreatesSystemPagesMenusAndSourceLanguageOnce(t *testing.T) {
	truncate(t, "pages", "menu_items", "languages", "document_templates")
	ctx := context.Background()
	require.NoError(t, seeders.Reference(ctx, testDB, noSettings{}, nil))
	exec(t, `DELETE FROM menu_items WHERE label = 'Gallery'`)
	require.NoError(t, seeders.Reference(ctx, testDB, noSettings{}, nil))

	page, err := repository.NewPageRepository(testDB).List(ctx, listAll(repository.ListQuery{Filters: map[string]any{"is_system": true}}))
	require.NoError(t, err)
	require.Equal(t, 5, page.Total)
	for _, p := range page.Items {
		assert.Equal(t, model.StatusPublished, p.Status, p.Title)
		assert.Equal(t, *p.SystemKey, p.Slug)
	}
	assert.Len(t, menuItemsOf(t, model.MenuHeader), 6, "default menus are only seeded on an empty menu table; About Us and Services are appended")
	assert.Len(t, menuItemsOf(t, model.MenuFooter), 5, "Terms and Privacy are appended to the default footer")

	langs, err := repository.NewLanguageRepository(testDB).List(ctx, listAll(repository.ListQuery{}))
	require.NoError(t, err)
	require.Equal(t, 2, langs.Total)
	assert.True(t, langs.Items[0].IsSource)
	assert.Equal(t, "en", langs.Items[0].Code)
}
