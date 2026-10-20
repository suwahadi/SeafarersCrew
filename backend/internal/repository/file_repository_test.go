package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

func TestFileRepositoryLifecycle(t *testing.T) {
	truncate(t, "files")
	ctx := context.Background()
	repo := repository.NewFileRepository(testDB)

	created, err := repo.Create(ctx, model.File{
		StorageKey: "public/cms_image/2026/10/abcdefghijkl.webp", OriginalName: "hero.jpg", MimeType: "image/webp",
		SizeOriginal: 2048, SizeStored: 512, SHA256: "ab", IsPublic: true, Status: model.FileReady,
		Variants: []model.FileVariant{{Width: 480, Height: 240, StorageKey: "public/cms_image/2026/10/abcdefghijkl-480.webp", Size: 100}},
	})
	require.NoError(t, err)
	assert.Empty(t, created.OwnerType)
	assert.Nil(t, created.OwnerID)

	got, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, got.Variants, 1)
	assert.Equal(t, 480, got.Variants[0].Width)

	owner := uuid.New()
	require.NoError(t, repo.AttachOwner(ctx, created.ID, "crews", owner))
	listed, err := repo.ListByOwner(ctx, "crews", owner)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, created.ID, listed[0].ID)

	require.NoError(t, repo.SoftDelete(ctx, created.ID))
	_, err = repo.Get(ctx, created.ID)
	assert.ErrorIs(t, err, repository.ErrNotFound)
	assert.ErrorIs(t, repo.SoftDelete(ctx, created.ID), repository.ErrNotFound)
	assert.ErrorIs(t, repo.AttachOwner(ctx, created.ID, "crews", owner), repository.ErrNotFound)

	listed, err = repo.ListByOwner(ctx, "crews", owner)
	require.NoError(t, err)
	assert.Empty(t, listed)
}

func TestFileStorageKeyIsUnique(t *testing.T) {
	truncate(t, "files")
	ctx := context.Background()
	repo := repository.NewFileRepository(testDB)
	f := model.File{StorageKey: "crews/2026/10/aaaaaaaaaaaa.pdf", OriginalName: "a.pdf", MimeType: "application/pdf", SHA256: "x", Status: model.FileReady}
	_, err := repo.Create(ctx, f)
	require.NoError(t, err)
	_, err = repo.Create(ctx, f)
	assert.Error(t, err)
}
