package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type memVacancies struct {
	rows          map[uuid.UUID]model.Vacancy
	activeRanks   map[uuid.UUID]bool
	activeVessels map[uuid.UUID]bool
	redirects     []model.Redirect
	idCollisions  int
	lastToday     time.Time
	openList      []model.Vacancy
}

func newMemVacancies() *memVacancies {
	return &memVacancies{rows: map[uuid.UUID]model.Vacancy{}, activeRanks: map[uuid.UUID]bool{}, activeVessels: map[uuid.UUID]bool{}}
}

func (m *memVacancies) Create(_ context.Context, v model.Vacancy, publicID string) (uuid.UUID, error) {
	if m.idCollisions > 0 {
		m.idCollisions--
		return uuid.Nil, &pgconn.PgError{Code: "23505", ConstraintName: "vacancies_public_id_key"}
	}
	for _, o := range m.rows {
		if o.Slug == v.Slug {
			return uuid.Nil, &pgconn.PgError{Code: "23505", ConstraintName: "vacancies_slug_key"}
		}
	}
	v.ID, v.PublicID = uuid.New(), publicID
	m.rows[v.ID] = v
	return v.ID, nil
}

func (m *memVacancies) Update(_ context.Context, v model.Vacancy, redirect *model.Redirect) error {
	if _, ok := m.rows[v.ID]; !ok {
		return repository.ErrNotFound
	}
	m.rows[v.ID] = v
	if redirect != nil {
		m.redirects = append(m.redirects, *redirect)
	}
	return nil
}

func (m *memVacancies) GetByPublicID(_ context.Context, publicID string) (model.Vacancy, error) {
	for _, v := range m.rows {
		if v.PublicID == publicID {
			return v, nil
		}
	}
	return model.Vacancy{}, repository.ErrNotFound
}

func (m *memVacancies) GetByID(_ context.Context, id uuid.UUID) (model.Vacancy, error) {
	v, ok := m.rows[id]
	if !ok {
		return v, repository.ErrNotFound
	}
	return v, nil
}

func (m *memVacancies) GetBySlug(_ context.Context, slug string) (model.Vacancy, error) {
	for _, v := range m.rows {
		if v.Slug == slug {
			return v, nil
		}
	}
	return model.Vacancy{}, repository.ErrNotFound
}

func (m *memVacancies) List(_ context.Context, _ model.VacancyFilter, _, _ int) (repository.Page[model.Vacancy], error) {
	var items []model.Vacancy
	for _, v := range m.rows {
		items = append(items, v)
	}
	return repository.Page[model.Vacancy]{Items: items, Total: len(items)}, nil
}

func (m *memVacancies) ListOpen(_ context.Context, f model.PublicVacancyFilter, _, _ int) (repository.Page[model.Vacancy], error) {
	m.lastToday = f.Today
	return repository.Page[model.Vacancy]{Items: m.openList, Total: len(m.openList)}, nil
}

func (m *memVacancies) FilterOptions(_ context.Context, today time.Time) (model.VacancyFilterOptions, error) {
	m.lastToday = today
	return model.VacancyFilterOptions{Locations: []string{"Singapore"}}, nil
}

func (m *memVacancies) SoftDelete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.rows[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.rows, id)
	return nil
}

func (m *memVacancies) SlugTaken(_ context.Context, slug string, exclude uuid.UUID) (bool, error) {
	for id, v := range m.rows {
		if v.Slug == slug && id != exclude {
			return true, nil
		}
	}
	return false, nil
}

func (m *memVacancies) RankActive(_ context.Context, id uuid.UUID) (bool, error) {
	return m.activeRanks[id], nil
}

func (m *memVacancies) VesselTypeActive(_ context.Context, id uuid.UUID) (bool, error) {
	return m.activeVessels[id], nil
}

type recordingPublisher struct {
	calls [][]string
	err   error
}

func (p *recordingPublisher) Changed(_ context.Context, tags ...string) error {
	p.calls = append(p.calls, tags)
	return p.err
}

type vacancyFixture struct {
	svc       *VacancyService
	store     *memVacancies
	publisher *recordingPublisher
	rank      uuid.UUID
	vessel    uuid.UUID
	now       time.Time
}

func newVacancyFixture(t *testing.T) *vacancyFixture {
	t.Helper()
	f := &vacancyFixture{store: newMemVacancies(), publisher: &recordingPublisher{}, rank: uuid.New(), vessel: uuid.New(),
		now: time.Date(2026, 10, 21, 3, 0, 0, 0, time.UTC)}
	f.store.activeRanks[f.rank] = true
	f.store.activeVessels[f.vessel] = true
	f.svc = NewVacancyService(f.store, staticSettings{"general": {"timezone": "Asia/Jakarta"}}, f.publisher, func() time.Time { return f.now })
	return f
}

func (f *vacancyFixture) input(title, status string) model.Vacancy {
	return model.Vacancy{Title: title, RankID: f.rank, Department: model.DepartmentDeck, EmploymentType: model.EmploymentContract, Status: status}
}

func day(y int, m time.Month, d int) *time.Time {
	t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return &t
}

func TestCreateVacancyGeneratesUniqueSlugAndPublicID(t *testing.T) {
	f := newVacancyFixture(t)
	ctx := context.Background()
	f.store.idCollisions = 2

	first, err := f.svc.Create(ctx, f.input("Chief Officer – Tanker", model.VacancyDraft))
	require.NoError(t, err)
	second, err := f.svc.Create(ctx, f.input("Chief Officer – Tanker", model.VacancyDraft))
	require.NoError(t, err)

	assert.Equal(t, "chief-officer-tanker", first.Slug)
	assert.Equal(t, "chief-officer-tanker-2", second.Slug)
	assert.Len(t, first.PublicID, 9)
	assert.Nil(t, first.PublishedAt, "drafts are not published")
	assert.Empty(t, f.publisher.calls, "draft changes do not touch the public cache")
}

func TestCreateVacancyUsesRequestedSlugNormalized(t *testing.T) {
	f := newVacancyFixture(t)
	in := f.input("Master", model.VacancyDraft)
	in.Slug = "  Master Mariner / LNG "

	v, err := f.svc.Create(context.Background(), in)

	require.NoError(t, err)
	assert.Equal(t, "master-mariner-lng", v.Slug)
}

func TestCreateVacancySanitizesRichText(t *testing.T) {
	f := newVacancyFixture(t)
	in := f.input("Master", model.VacancyDraft)
	in.DescriptionHTML = `<p onclick="x()">Lead the <strong>crew</strong></p><script>alert(1)</script>`
	in.RequirementsHTML = `<ul><li>STCW</li></ul><iframe src="https://evil.example"></iframe>`

	v, err := f.svc.Create(context.Background(), in)

	require.NoError(t, err)
	assert.Equal(t, `<p>Lead the <strong>crew</strong></p>`, v.DescriptionHTML)
	assert.Equal(t, `<ul><li>STCW</li></ul>`, v.RequirementsHTML)
}

func TestCreateVacancyRequiresActiveReferences(t *testing.T) {
	f := newVacancyFixture(t)
	missingVessel := uuid.New()
	tests := []struct {
		name   string
		mutate func(*model.Vacancy)
		field  string
	}{
		{"unknown rank", func(v *model.Vacancy) { v.RankID = uuid.New() }, "rank_id"},
		{"inactive vessel type", func(v *model.Vacancy) { v.VesselTypeID = &missingVessel }, "vessel_type_id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := f.input("Master", model.VacancyDraft)
			tc.mutate(&in)
			_, err := f.svc.Create(context.Background(), in)
			var e *apperr.Error
			require.ErrorAs(t, err, &e)
			assert.Contains(t, e.Fields, tc.field)
		})
	}
	assert.Empty(t, f.store.rows)
}

func TestOpeningVacancyRequiresClosingDateTodayOrLater(t *testing.T) {
	f := newVacancyFixture(t)
	ctx := context.Background()
	tests := []struct {
		name    string
		status  string
		closing *time.Time
		ok      bool
	}{
		{"open without closing date", model.VacancyOpen, nil, true},
		{"open closing today in the settings timezone", model.VacancyOpen, day(2026, 10, 21), true},
		{"open closing yesterday", model.VacancyOpen, day(2026, 10, 20), false},
		{"draft may keep a past closing date", model.VacancyDraft, day(2026, 10, 1), true},
		{"closed may keep a past closing date", model.VacancyClosed, day(2026, 10, 1), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := f.input(tc.name, tc.status)
			in.ClosingDate = tc.closing
			_, err := f.svc.Create(ctx, in)
			if tc.ok {
				assert.NoError(t, err)
				return
			}
			var e *apperr.Error
			require.ErrorAs(t, err, &e)
			assert.Contains(t, e.Fields, "closing_date")
		})
	}
}

func TestAlreadyOpenVacancyKeepsItsPastClosingDateWhenEdited(t *testing.T) {
	f := newVacancyFixture(t)
	ctx := context.Background()
	created, err := f.svc.Create(ctx, withClosing(f.input("Master", model.VacancyOpen), day(2026, 10, 25)))
	require.NoError(t, err)
	f.now = time.Date(2026, 11, 2, 3, 0, 0, 0, time.UTC)

	edited := withClosing(f.input("Master edited", model.VacancyOpen), day(2026, 10, 25))
	got, err := f.svc.Update(ctx, created.PublicID, edited)
	require.NoError(t, err)
	assert.True(t, got.IsExpired, "an open vacancy past its closing date is reported as expired")

	moved := withClosing(f.input("Master edited", model.VacancyOpen), day(2026, 10, 30))
	_, err = f.svc.Update(ctx, created.PublicID, moved)
	assert.Error(t, err, "moving the closing date into the past is refused")
}

func withClosing(v model.Vacancy, d *time.Time) model.Vacancy {
	v.ClosingDate = d
	return v
}

func TestVacancyPublishedAtIsSetOnFirstOpenOnly(t *testing.T) {
	f := newVacancyFixture(t)
	ctx := context.Background()
	created, err := f.svc.Create(ctx, f.input("Master", model.VacancyDraft))
	require.NoError(t, err)
	require.Nil(t, created.PublishedAt)

	opened, err := f.svc.SetStatus(ctx, created.PublicID, model.VacancyOpen)
	require.NoError(t, err)
	require.NotNil(t, opened.PublishedAt)
	first := *opened.PublishedAt
	assert.Equal(t, f.now, first)

	f.now = f.now.Add(48 * time.Hour)
	closed, err := f.svc.SetStatus(ctx, created.PublicID, model.VacancyClosed)
	require.NoError(t, err)
	reopened, err := f.svc.SetStatus(ctx, created.PublicID, model.VacancyOpen)
	require.NoError(t, err)
	assert.Equal(t, first, *closed.PublishedAt)
	assert.Equal(t, first, *reopened.PublishedAt, "reopening keeps the first publication date")
}

func TestChangingSlugOfPublishedVacancyStoresRedirectAndBustsRedirectCache(t *testing.T) {
	f := newVacancyFixture(t)
	ctx := context.Background()
	created, err := f.svc.Create(ctx, f.input("Master", model.VacancyOpen))
	require.NoError(t, err)
	f.publisher.calls = nil

	in := f.input("Master", model.VacancyOpen)
	in.Slug = "master-mariner"
	got, err := f.svc.Update(ctx, created.PublicID, in)

	require.NoError(t, err)
	assert.Equal(t, "master-mariner", got.Slug)
	assert.Equal(t, []model.Redirect{{FromPath: "/career/master", ToPath: "/career/master-mariner"}}, f.store.redirects)
	require.Len(t, f.publisher.calls, 1)
	assert.ElementsMatch(t, []string{model.TagVacancies, model.TagHome, model.TagSitemap, model.TagRedirects}, f.publisher.calls[0])
}

func TestChangingSlugOfNeverPublishedVacancyStoresNoRedirect(t *testing.T) {
	f := newVacancyFixture(t)
	ctx := context.Background()
	created, err := f.svc.Create(ctx, f.input("Master", model.VacancyDraft))
	require.NoError(t, err)

	in := f.input("Master", model.VacancyDraft)
	in.Slug = "master-mariner"
	_, err = f.svc.Update(ctx, created.PublicID, in)

	require.NoError(t, err)
	assert.Empty(t, f.store.redirects)
}

func TestUpdateVacancyKeepsSlugWhenNoneIsSent(t *testing.T) {
	f := newVacancyFixture(t)
	ctx := context.Background()
	created, err := f.svc.Create(ctx, f.input("Master", model.VacancyDraft))
	require.NoError(t, err)

	got, err := f.svc.Update(ctx, created.PublicID, f.input("A new title", model.VacancyDraft))

	require.NoError(t, err)
	assert.Equal(t, "master", got.Slug, "renaming does not regenerate the slug")
}

func TestUpdateVacancyRejectsSlugOfAnotherVacancy(t *testing.T) {
	f := newVacancyFixture(t)
	ctx := context.Background()
	_, err := f.svc.Create(ctx, f.input("Master", model.VacancyDraft))
	require.NoError(t, err)
	other, err := f.svc.Create(ctx, f.input("Engineer", model.VacancyDraft))
	require.NoError(t, err)

	in := f.input("Engineer", model.VacancyDraft)
	in.Slug = "master"
	got, err := f.svc.Update(ctx, other.PublicID, in)

	require.NoError(t, err)
	assert.Equal(t, "master-2", got.Slug, "a taken slug gets a numeric suffix")
}

func TestVacancyLookupWithMalformedIDIsNotFound(t *testing.T) {
	f := newVacancyFixture(t)
	for _, id := range []string{"", "short", "ABCDEFGH0", "abc"} {
		_, err := f.svc.Get(context.Background(), id)
		var e *apperr.Error
		require.ErrorAs(t, err, &e)
		assert.Equal(t, 404, e.Status)
	}
}

func TestDeleteVacancyIsSoftAndBustsPublicCacheOnlyWhenPublished(t *testing.T) {
	f := newVacancyFixture(t)
	ctx := context.Background()
	draft, err := f.svc.Create(ctx, f.input("Draft", model.VacancyDraft))
	require.NoError(t, err)
	open, err := f.svc.Create(ctx, f.input("Open", model.VacancyOpen))
	require.NoError(t, err)
	f.publisher.calls = nil

	require.NoError(t, f.svc.Delete(ctx, draft.PublicID))
	assert.Empty(t, f.publisher.calls)
	require.NoError(t, f.svc.Delete(ctx, open.PublicID))
	assert.Len(t, f.publisher.calls, 1)
	assert.Error(t, f.svc.Delete(ctx, open.PublicID))
}

func TestVacancyWriteSucceedsWhenCacheInvalidationFails(t *testing.T) {
	f := newVacancyFixture(t)
	f.publisher.err = assert.AnError

	v, err := f.svc.Create(context.Background(), f.input("Master", model.VacancyOpen))

	require.NoError(t, err)
	assert.Equal(t, model.VacancyOpen, v.Status)
}

func TestPublicDetailReportsClosedAndExpiredVacanciesAsNotOpen(t *testing.T) {
	f := newVacancyFixture(t)
	ctx := context.Background()
	past, today := day(2026, 10, 20), day(2026, 10, 21)
	seed := func(slug, status string, closing *time.Time) {
		id := uuid.New()
		f.store.rows[id] = model.Vacancy{ID: id, PublicID: "VAC22222" + slug[:1], Slug: slug, Status: status, ClosingDate: closing}
	}
	seed("open", model.VacancyOpen, nil)
	seed("last-day", model.VacancyOpen, today)
	seed("expired", model.VacancyOpen, past)
	seed("closed", model.VacancyClosed, nil)
	seed("draft", model.VacancyDraft, nil)

	tests := []struct {
		slug   string
		isOpen bool
		found  bool
	}{
		{"open", true, true},
		{"last-day", true, true},
		{"expired", false, true},
		{"closed", false, true},
		{"draft", false, false},
		{"missing", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.slug, func(t *testing.T) {
			v, err := f.svc.PublicDetail(ctx, tc.slug)
			if !tc.found {
				var e *apperr.Error
				require.ErrorAs(t, err, &e)
				assert.Equal(t, 404, e.Status)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.isOpen, v.IsOpen)
		})
	}
}

func TestPublicListUsesTodayInTheSettingsTimezone(t *testing.T) {
	f := newVacancyFixture(t)
	f.now = time.Date(2026, 10, 20, 20, 0, 0, 0, time.UTC)

	_, err := f.svc.PublicList(context.Background(), model.PublicVacancyQuery{Page: 1, PerPage: 12})
	require.NoError(t, err)
	assert.Equal(t, *day(2026, 10, 21), f.store.lastToday, "20:00 UTC is already the next day in Jakarta")

	_, err = f.svc.PublicFilters(context.Background())
	require.NoError(t, err)
	assert.Equal(t, *day(2026, 10, 21), f.store.lastToday)
}
