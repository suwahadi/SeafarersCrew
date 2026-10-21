package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

func TestSaveRedirectFlattensChainsAndClearsLivePaths(t *testing.T) {
	truncate(t, "redirects")
	ctx := context.Background()
	repo := repository.NewRedirectRepository(testDB)

	require.NoError(t, repository.SaveRedirect(ctx, testDB, "/about-us", "/about"))
	require.NoError(t, repository.SaveRedirect(ctx, testDB, "/about", "/company"))

	rd, err := repo.Resolve(ctx, "/about-us")
	require.NoError(t, err)
	assert.Equal(t, "/company", rd.ToPath, "old redirects point straight at the newest path")
	assert.Equal(t, 301, rd.StatusCode)

	require.NoError(t, repository.SaveRedirect(ctx, testDB, "/company", "/about-us"))
	_, err = repo.Resolve(ctx, "/about-us")
	assert.ErrorIs(t, err, repository.ErrNotFound, "a path that is live again no longer redirects")
	rd, err = repo.Resolve(ctx, "/about")
	require.NoError(t, err)
	assert.Equal(t, "/about-us", rd.ToPath)
}

func TestSlugTakenIgnoresSelfAndDeletedRows(t *testing.T) {
	truncate(t, "news")
	ctx := context.Background()
	id := mustID(t, `INSERT INTO news (title, slug) VALUES ('A', 'crew-change') RETURNING id`)
	exec(t, `INSERT INTO news (title, slug, deleted_at) VALUES ('B', 'old-story', now())`)

	taken, err := repository.SlugTaken(ctx, testDB, "news", "crew-change", uuid.Nil)
	require.NoError(t, err)
	assert.True(t, taken)
	taken, err = repository.SlugTaken(ctx, testDB, "news", "crew-change", id)
	require.NoError(t, err)
	assert.False(t, taken)
	taken, err = repository.SlugTaken(ctx, testDB, "news", "old-story", uuid.Nil)
	require.NoError(t, err)
	assert.False(t, taken)

	_, err = repository.SlugTaken(ctx, testDB, "crews", "x", uuid.Nil)
	assert.Error(t, err)
}
