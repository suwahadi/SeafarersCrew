package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type memList[T any] struct {
	rows    map[uuid.UUID]T
	setID   func(*T, uuid.UUID)
	order   []uuid.UUID
	codeOf  func(T) string
	deleted []uuid.UUID
}

func newMemList[T any](setID func(*T, uuid.UUID), codeOf func(T) string) *memList[T] {
	return &memList[T]{rows: map[uuid.UUID]T{}, setID: setID, codeOf: codeOf}
}

func (m *memList[T]) List(context.Context, repository.ListQuery) (repository.Page[T], error) {
	return repository.Page[T]{}, nil
}

func (m *memList[T]) Get(_ context.Context, id uuid.UUID) (T, error) {
	v, ok := m.rows[id]
	if !ok {
		return v, repository.ErrNotFound
	}
	return v, nil
}

func (m *memList[T]) Insert(_ context.Context, v T) (T, error) {
	id := uuid.New()
	m.setID(&v, id)
	m.rows[id] = v
	return v, nil
}

func (m *memList[T]) InsertAll(ctx context.Context, items []T) ([]T, error) {
	out := make([]T, 0, len(items))
	for _, it := range items {
		v, _ := m.Insert(ctx, it)
		out = append(out, v)
	}
	return out, nil
}

func (m *memList[T]) Update(_ context.Context, id uuid.UUID, v T) (T, error) {
	m.setID(&v, id)
	m.rows[id] = v
	return v, nil
}

func (m *memList[T]) SoftDelete(_ context.Context, id uuid.UUID) error {
	m.deleted = append(m.deleted, id)
	delete(m.rows, id)
	return nil
}

func (m *memList[T]) Reorder(_ context.Context, ids []uuid.UUID) error {
	m.order = ids
	return nil
}

func (m *memList[T]) Taken(_ context.Context, _, value string, exclude uuid.UUID) (bool, error) {
	for id, v := range m.rows {
		if id != exclude && m.codeOf != nil && m.codeOf(v) == value {
			return true, nil
		}
	}
	return false, nil
}

func (m *memList[T]) Usages(context.Context, uuid.UUID) ([]model.Usage, error) { return nil, nil }

type cmsFixture struct {
	cms       CMS
	languages *memList[model.Language]
	gallery   *memList[model.GalleryImage]
	slides    *memList[model.HeroSlide]
	files     *fakeFileLookup
	site      *siteRecorder
}

func newCMSFixture() cmsFixture {
	f := cmsFixture{
		languages: newMemList(func(l *model.Language, id uuid.UUID) { l.ID = id }, func(l model.Language) string { return l.Code }),
		gallery:   newMemList(func(g *model.GalleryImage, id uuid.UUID) { g.ID = id }, nil),
		slides:    newMemList(func(s *model.HeroSlide, id uuid.UUID) { s.ID = id }, nil),
		files:     newFakeFiles(),
		site:      &siteRecorder{},
	}
	f.cms = NewCMS(CMSStores{
		HeroSlides:   f.slides,
		Gallery:      f.gallery,
		Testimonials: newMemList(func(t *model.Testimonial, id uuid.UUID) { t.ID = id }, nil),
		ClientLogos:  newMemList(func(c *model.ClientLogo, id uuid.UUID) { c.ID = id }, nil),
		Stats:        newMemList(func(s *model.StatsItem, id uuid.UUID) { s.ID = id }, nil),
		Languages:    f.languages,
	}, NewPublicFileChecker(f.files), f.site)
	return f
}

func TestHeroSlidesRequireReadyPublicImages(t *testing.T) {
	f := newCMSFixture()
	ctx := context.Background()
	webp := f.files.add("image/webp", "a.jpg")
	_, err := f.cms.HeroSlides.Create(ctx, model.HeroSlide{ImageFileID: uuid.New()})
	assert.Contains(t, validationFields(t, err), "image_file_id")

	svg := f.files.add("image/svg+xml", "a.svg")
	_, err = f.cms.HeroSlides.Create(ctx, model.HeroSlide{ImageFileID: webp, MobileImageFileID: &svg})
	assert.Contains(t, validationFields(t, err), "mobile_image_file_id")

	_, err = f.cms.HeroSlides.Create(ctx, model.HeroSlide{ImageFileID: webp, MobileImageFileID: &webp})
	require.NoError(t, err)
	assert.Equal(t, []string{model.TagHome}, f.site.last())
}

func TestClientLogosAcceptWebPAndSVGButNotMissingFiles(t *testing.T) {
	f := newCMSFixture()
	ctx := context.Background()
	for _, mime := range []string{"image/webp", "image/svg+xml"} {
		_, err := f.cms.ClientLogos.Create(ctx, model.ClientLogo{Name: "Acme", LogoFileID: f.files.add(mime, "logo")})
		require.NoError(t, err, mime)
	}
	_, err := f.cms.ClientLogos.Create(ctx, model.ClientLogo{Name: "Acme", LogoFileID: uuid.New()})
	assert.Contains(t, validationFields(t, err), "logo_file_id")
}

func TestTestimonialPhotoIsOptionalButCheckedWhenGiven(t *testing.T) {
	f := newCMSFixture()
	ctx := context.Background()
	_, err := f.cms.Testimonials.Create(ctx, model.Testimonial{Name: "Ari", Quote: "q", Rating: 5})
	require.NoError(t, err)
	missing := uuid.New()
	_, err = f.cms.Testimonials.Create(ctx, model.Testimonial{Name: "Ari", Quote: "q", Rating: 5, PhotoFileID: &missing})
	assert.Contains(t, validationFields(t, err), "photo_file_id")
}

func TestLanguagesNeedUniqueCodesAndSVGFlags(t *testing.T) {
	f := newCMSFixture()
	ctx := context.Background()
	_, err := f.cms.Languages.Create(ctx, model.Language{Name: "Japanese", Code: "ja", FlagCode: "jp", MasterBase: model.MasterBase{IsActive: true}})
	require.NoError(t, err)
	assert.Equal(t, []string{model.TagSite}, f.site.last())

	_, err = f.cms.Languages.Create(ctx, model.Language{Name: "Japanese again", Code: "ja", FlagCode: "jp"})
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, 409, e.Status)
	assert.Contains(t, e.Fields, "code")

	webp := f.files.add("image/webp", "flag.webp")
	_, err = f.cms.Languages.Create(ctx, model.Language{Name: "Korean", Code: "ko", FlagFileID: &webp})
	assert.Contains(t, validationFields(t, err), "flag_file_id")
	svg := f.files.add("image/svg+xml", "flag.svg")
	_, err = f.cms.Languages.Create(ctx, model.Language{Name: "Korean", Code: "ko", FlagFileID: &svg})
	require.NoError(t, err)
}

func TestTheSourceLanguageCannotBeDeletedDeactivatedOrRecoded(t *testing.T) {
	f := newCMSFixture()
	ctx := context.Background()
	id := uuid.New()
	f.languages.rows[id] = model.Language{MasterBase: model.MasterBase{ID: id, IsActive: true}, Name: "English", Code: "en", FlagCode: "gb", IsSource: true}

	err := f.cms.Languages.Delete(ctx, id)
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, "source_language", e.Code)
	assert.Empty(t, f.languages.deleted)

	_, err = f.cms.Languages.Update(ctx, id, model.Language{Name: "English", Code: "en", FlagCode: "gb", MasterBase: model.MasterBase{IsActive: false}})
	assert.Contains(t, validationFields(t, err), "is_active")

	_, err = f.cms.Languages.Update(ctx, id, model.Language{Name: "English", Code: "eng", FlagCode: "gb", MasterBase: model.MasterBase{IsActive: true}})
	assert.Contains(t, validationFields(t, err), "code")

	updated, err := f.cms.Languages.Update(ctx, id, model.Language{Name: "English (UK)", Code: "EN", FlagCode: "gb", MasterBase: model.MasterBase{IsActive: true}})
	require.NoError(t, err)
	assert.Equal(t, "English (UK)", updated.Name)
	assert.True(t, updated.IsActive)
}

func TestOtherLanguagesCanBeDeactivatedAndDeleted(t *testing.T) {
	f := newCMSFixture()
	ctx := context.Background()
	created, err := f.cms.Languages.Create(ctx, model.Language{Name: "Japanese", Code: "ja", FlagCode: "jp", MasterBase: model.MasterBase{IsActive: true}})
	require.NoError(t, err)
	off, err := f.cms.Languages.Update(ctx, created.ID, model.Language{Name: "Japanese", Code: "ja", FlagCode: "jp"})
	require.NoError(t, err)
	assert.False(t, off.IsActive)
	require.NoError(t, f.cms.Languages.Delete(ctx, created.ID))
	assert.Equal(t, []uuid.UUID{created.ID}, f.languages.deleted)
}

func TestGalleryBulkCreateFillsMissingAltTextFromCaptionThenFileName(t *testing.T) {
	f := newCMSFixture()
	ctx := context.Background()
	a := f.files.add("image/webp", "crew-on_deck.JPG")
	b := f.files.add("image/webp", "bridge.png")
	c := f.files.add("image/webp", "x.png")

	created, err := f.cms.Gallery.BulkCreate(ctx, []model.GalleryImage{
		{ImageFileID: a},
		{ImageFileID: b, Caption: "The bridge"},
		{ImageFileID: c, AltText: "Given alt"},
	})
	require.NoError(t, err)
	require.Len(t, created, 3)
	assert.Equal(t, "crew on deck", created[0].AltText)
	assert.Equal(t, "The bridge", created[1].AltText)
	assert.Equal(t, "Given alt", created[2].AltText)
	assert.ElementsMatch(t, []string{model.TagGallery, model.TagHome, model.TagPages}, f.site.last())
}

func TestGalleryBulkCreateIsAllOrNothing(t *testing.T) {
	f := newCMSFixture()
	ctx := context.Background()
	good := f.files.add("image/webp", "a.jpg")

	_, err := f.cms.Gallery.BulkCreate(ctx, []model.GalleryImage{{ImageFileID: good, AltText: "ok"}, {ImageFileID: uuid.New(), AltText: "bad"}})
	assert.Contains(t, validationFields(t, err), "items.1.image_file_id")
	assert.Empty(t, f.gallery.rows)

	_, err = f.cms.Gallery.BulkCreate(ctx, nil)
	assert.Contains(t, validationFields(t, err), "items")

	tooMany := make([]model.GalleryImage, MaxBulkGalleryItems+1)
	_, err = f.cms.Gallery.BulkCreate(ctx, tooMany)
	assert.Contains(t, validationFields(t, err), "items")
}

func TestGalleryChangesAnnounceGalleryHomeAndPages(t *testing.T) {
	f := newCMSFixture()
	img := f.files.add("image/webp", "a.jpg")
	_, err := f.cms.Gallery.Create(context.Background(), model.GalleryImage{ImageFileID: img, AltText: "a"})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{model.TagGallery, model.TagHome, model.TagPages}, f.site.last())
}

func TestStatsAndReorderAnnounceHome(t *testing.T) {
	f := newCMSFixture()
	ctx := context.Background()
	s, err := f.cms.Stats.Create(ctx, model.StatsItem{Number: 5, Label: "Ships"})
	require.NoError(t, err)
	assert.Equal(t, []string{model.TagHome}, f.site.last())
	before := f.site.count()
	require.NoError(t, f.cms.Stats.Reorder(ctx, []uuid.UUID{s.ID}))
	assert.Equal(t, before+1, f.site.count())
}
