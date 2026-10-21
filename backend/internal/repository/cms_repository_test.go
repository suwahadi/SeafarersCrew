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

func publicFile(t *testing.T, mime string) uuid.UUID {
	t.Helper()
	key := "public/cms_image/2026/10/" + uuid.NewString() + ".webp"
	return mustID(t, `INSERT INTO files (storage_key, original_name, mime_type, size_original, size_stored, sha256, is_public, width, height, variants)
		VALUES ($1, 'Crew Photo.jpg', $2, 100, 80, 'x', true, 1600, 900,
		'[{"width":480,"height":270,"storage_key":"`+key+`-480","size":10}]'::jsonb) RETURNING id`, key, mime)
}

func TestHeroSlideRepositoryReturnsImagesAndAppendsToTheEnd(t *testing.T) {
	truncate(t, "hero_slides", "files")
	ctx := context.Background()
	repo := repository.NewHeroSlideRepository(testDB)
	desktop := publicFile(t, "image/webp")
	mobile := publicFile(t, "image/webp")

	first, err := repo.Insert(ctx, model.HeroSlide{MasterBase: model.MasterBase{IsActive: true}, ImageFileID: desktop, MobileImageFileID: &mobile, Heading: "Crew"})
	require.NoError(t, err)
	second, err := repo.Insert(ctx, model.HeroSlide{MasterBase: model.MasterBase{IsActive: false}, ImageFileID: desktop, Heading: "Fleet"})
	require.NoError(t, err)

	assert.Equal(t, 10, first.SortOrder)
	assert.Equal(t, 20, second.SortOrder)
	require.NotNil(t, first.Image)
	assert.Equal(t, desktop, first.Image.ID)
	assert.Len(t, first.Image.Variants, 1)
	require.NotNil(t, first.MobileImage)
	assert.Nil(t, second.MobileImage)

	active, err := repo.Active(ctx)
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.Equal(t, "Crew", active[0].Heading)

	require.NoError(t, repo.Reorder(ctx, []uuid.UUID{second.ID, first.ID}))
	page, err := repo.List(ctx, listAll(repository.ListQuery{}))
	require.NoError(t, err)
	assert.Equal(t, "Fleet", page.Items[0].Heading)
}

func TestPrivateFilesAreNotExposedAsImages(t *testing.T) {
	truncate(t, "gallery_images", "files")
	ctx := context.Background()
	privateID := mustID(t, `INSERT INTO files (storage_key, original_name, mime_type, size_original, size_stored, sha256)
		VALUES ('crew_photo/2026/10/p.webp', 'p.jpg', 'image/webp', 1, 1, 'x') RETURNING id`)
	g, err := repository.NewGalleryImageRepository(testDB).Insert(ctx, model.GalleryImage{MasterBase: model.MasterBase{IsActive: true}, ImageFileID: privateID, AltText: "x"})
	require.NoError(t, err)
	assert.Nil(t, g.Image)

	lookup := repository.NewCMSLookupRepository(testDB)
	infos, err := lookup.PublicFiles(ctx, []uuid.UUID{privateID})
	require.NoError(t, err)
	assert.Empty(t, infos)
}

func TestCMSLookupFindsPublicFilesAndImages(t *testing.T) {
	truncate(t, "files")
	ctx := context.Background()
	lookup := repository.NewCMSLookupRepository(testDB)
	ready := publicFile(t, "image/webp")
	processing := mustID(t, `INSERT INTO files (storage_key, original_name, mime_type, size_original, size_stored, sha256, is_public, status)
		VALUES ('public/cms_image/2026/10/z.webp', 'z.jpg', 'image/webp', 1, 1, 'x', true, 'processing') RETURNING id`)
	deleted := mustID(t, `INSERT INTO files (storage_key, original_name, mime_type, size_original, size_stored, sha256, is_public, deleted_at)
		VALUES ('public/cms_image/2026/10/d.webp', 'd.jpg', 'image/webp', 1, 1, 'x', true, now()) RETURNING id`)

	infos, err := lookup.PublicFiles(ctx, []uuid.UUID{ready, processing, deleted, uuid.New()})
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, "Crew Photo.jpg", infos[ready].OriginalName)

	images, err := lookup.Images(ctx, []uuid.UUID{ready, deleted})
	require.NoError(t, err)
	require.Len(t, images, 1)
	assert.Equal(t, 1600, images[ready].Width)

	empty, err := lookup.Images(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestGalleryInsertAllIsAtomic(t *testing.T) {
	truncate(t, "gallery_images", "files")
	ctx := context.Background()
	repo := repository.NewGalleryImageRepository(testDB)
	file := publicFile(t, "image/webp")

	created, err := repo.InsertAll(ctx, []model.GalleryImage{
		{MasterBase: model.MasterBase{IsActive: true}, ImageFileID: file, AltText: "one"},
		{MasterBase: model.MasterBase{IsActive: true}, ImageFileID: file, AltText: "two", ShowOnHome: true},
	})
	require.NoError(t, err)
	require.Len(t, created, 2)
	assert.Equal(t, []int{10, 20}, []int{created[0].SortOrder, created[1].SortOrder})

	_, err = repo.InsertAll(ctx, []model.GalleryImage{
		{MasterBase: model.MasterBase{IsActive: true}, ImageFileID: file, AltText: "three"},
		{MasterBase: model.MasterBase{IsActive: true}, ImageFileID: file, AltText: ""},
	})
	require.Error(t, err, "the alt text check constraint rejects the second row")
	page, err := repo.List(ctx, listAll(repository.ListQuery{}))
	require.NoError(t, err)
	assert.Equal(t, 2, page.Total, "a failed bulk insert leaves nothing behind")

	page, err = repo.List(ctx, listAll(repository.ListQuery{Filters: map[string]any{"show_on_home": true}}))
	require.NoError(t, err)
	assert.Equal(t, 1, page.Total)
}

func TestLanguageRepositoryAllowsReusingTheCodeOfADeletedLanguage(t *testing.T) {
	truncate(t, "languages")
	ctx := context.Background()
	repo := repository.NewLanguageRepository(testDB)

	ja, err := repo.Insert(ctx, model.Language{MasterBase: model.MasterBase{IsActive: true}, Name: "Chinese", Code: "zh-CN", FlagCode: "cn"})
	require.NoError(t, err)
	_, err = repo.Insert(ctx, model.Language{MasterBase: model.MasterBase{IsActive: true}, Name: "Chinese again", Code: "zh-cn", FlagCode: "cn"})
	require.ErrorIs(t, err, repository.ErrDuplicate, "codes are unique ignoring case")

	taken, err := repo.Taken(ctx, "code", "zh-cn", uuid.Nil)
	require.NoError(t, err)
	assert.True(t, taken)

	require.NoError(t, repo.SoftDelete(ctx, ja.ID))
	_, err = repo.Insert(ctx, model.Language{MasterBase: model.MasterBase{IsActive: true}, Name: "Chinese", Code: "zh-CN", FlagCode: "cn"})
	require.NoError(t, err)
}

func TestTestimonialAndClientLogoRepositoriesRoundTrip(t *testing.T) {
	truncate(t, "testimonials", "client_logos", "stats_items", "files")
	ctx := context.Background()
	file := publicFile(t, "image/svg+xml")

	testimonials := repository.NewTestimonialRepository(testDB)
	tm, err := testimonials.Insert(ctx, model.Testimonial{MasterBase: model.MasterBase{IsActive: true}, Name: "Capt. Ari", Position: "Master", PhotoFileID: &file, Quote: "Great crew.", Rating: 5})
	require.NoError(t, err)
	assert.Equal(t, 5, tm.Rating)
	require.NotNil(t, tm.Photo)
	_, err = testimonials.Insert(ctx, model.Testimonial{MasterBase: model.MasterBase{IsActive: true}, Name: "Bad", Quote: "x", Rating: 9})
	assert.Error(t, err, "rating is limited to 1 to 5")
	page, err := testimonials.List(ctx, listAll(repository.ListQuery{Q: "master"}))
	require.NoError(t, err)
	assert.Equal(t, 1, page.Total)

	logos := repository.NewClientLogoRepository(testDB)
	logo, err := logos.Insert(ctx, model.ClientLogo{MasterBase: model.MasterBase{IsActive: true}, Name: "Acme", LogoFileID: file, URL: "https://acme.example.com"})
	require.NoError(t, err)
	require.NotNil(t, logo.Logo)
	assert.Equal(t, "image/svg+xml", logo.Logo.MimeType)

	stats := repository.NewStatsItemRepository(testDB)
	stat, err := stats.Insert(ctx, model.StatsItem{MasterBase: model.MasterBase{IsActive: true}, Number: 1200, Suffix: "+", Label: "Seafarers Deployed"})
	require.NoError(t, err)
	assert.Equal(t, 1200, stat.Number)
}
