package service

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

// memMaster is an in-memory MasterStore; key extracts the unique value checked by Taken.
type memMaster[T any] struct {
	items   map[uuid.UUID]T
	setID   func(*T, uuid.UUID)
	key     func(T, string) string
	usages  map[uuid.UUID][]model.Usage
	deleted []uuid.UUID
	order   []uuid.UUID
}

func newMemMaster[T any](setID func(*T, uuid.UUID), key func(T, string) string) *memMaster[T] {
	return &memMaster[T]{items: map[uuid.UUID]T{}, setID: setID, key: key, usages: map[uuid.UUID][]model.Usage{}}
}

func (m *memMaster[T]) List(context.Context, repository.ListQuery) (repository.Page[T], error) {
	var out []T
	for _, it := range m.items {
		out = append(out, it)
	}
	return repository.Page[T]{Items: out, Total: len(out)}, nil
}

func (m *memMaster[T]) Get(_ context.Context, id uuid.UUID) (T, error) {
	it, ok := m.items[id]
	if !ok {
		var zero T
		return zero, repository.ErrNotFound
	}
	return it, nil
}

func (m *memMaster[T]) Insert(_ context.Context, item T) (T, error) {
	id := uuid.New()
	m.setID(&item, id)
	m.items[id] = item
	return item, nil
}

func (m *memMaster[T]) Update(_ context.Context, id uuid.UUID, item T) (T, error) {
	m.setID(&item, id)
	m.items[id] = item
	return item, nil
}

func (m *memMaster[T]) SoftDelete(_ context.Context, id uuid.UUID) error {
	delete(m.items, id)
	m.deleted = append(m.deleted, id)
	return nil
}

func (m *memMaster[T]) Reorder(_ context.Context, ids []uuid.UUID) error {
	m.order = ids
	return nil
}

func (m *memMaster[T]) Taken(_ context.Context, field, value string, exclude uuid.UUID) (bool, error) {
	for id, it := range m.items {
		if id != exclude && strings.EqualFold(m.key(it, field), value) {
			return true, nil
		}
	}
	return false, nil
}

func (m *memMaster[T]) Usages(_ context.Context, id uuid.UUID) ([]model.Usage, error) {
	return m.usages[id], nil
}

type fakeLookup struct {
	companies   map[uuid.UUID]string
	vesselTypes map[uuid.UUID]bool
	files       map[uuid.UUID]bool
}

func (f fakeLookup) CompanyType(_ context.Context, id uuid.UUID) (string, error) {
	t, ok := f.companies[id]
	if !ok {
		return "", repository.ErrNotFound
	}
	return t, nil
}

func (f fakeLookup) VesselTypeExists(_ context.Context, id uuid.UUID) (bool, error) {
	return f.vesselTypes[id], nil
}

func (f fakeLookup) FileExists(_ context.Context, id uuid.UUID) (bool, error) {
	return f.files[id], nil
}

type masterFixture struct {
	data       MasterData
	vessels    *memMaster[model.Vessel]
	companies  *memMaster[model.Company]
	statuses   *memMaster[model.CrewStatus]
	currencies *memMaster[model.Currency]
	ranks      *memMaster[model.Rank]
	docTypes   *memMaster[model.DocumentType]
	lookup     fakeLookup
	changes    *int
}

func newMasterFixture() masterFixture {
	f := masterFixture{
		vessels: newMemMaster(func(v *model.Vessel, id uuid.UUID) { v.ID = id }, func(v model.Vessel, _ string) string { return v.IMONumber }),
		companies: newMemMaster(func(c *model.Company, id uuid.UUID) { c.ID = id },
			func(c model.Company, _ string) string { return c.Code }),
		statuses: newMemMaster(func(s *model.CrewStatus, id uuid.UUID) { s.ID = id },
			func(s model.CrewStatus, _ string) string { return s.Name }),
		currencies: newMemMaster(func(c *model.Currency, id uuid.UUID) { c.ID = id },
			func(c model.Currency, _ string) string { return c.Code }),
		ranks: newMemMaster(func(r *model.Rank, id uuid.UUID) { r.ID = id },
			func(r model.Rank, _ string) string { return r.ShortCode }),
		docTypes: newMemMaster(func(d *model.DocumentType, id uuid.UUID) { d.ID = id },
			func(d model.DocumentType, _ string) string { return d.Name }),
		lookup:  fakeLookup{companies: map[uuid.UUID]string{}, vesselTypes: map[uuid.UUID]bool{}, files: map[uuid.UUID]bool{}},
		changes: new(int),
	}
	stores := MasterStores{
		Companies: f.companies, Vessels: f.vessels, Ranks: f.ranks, CrewStatuses: f.statuses, Currencies: f.currencies, DocumentTypes: f.docTypes,
		VesselTypes: newMemMaster(func(v *model.VesselType, id uuid.UUID) { v.ID = id }, func(v model.VesselType, _ string) string { return v.Name }),
		AllowanceTypes: newMemMaster(func(a *model.AllowanceType, id uuid.UUID) { a.ID = id },
			func(a model.AllowanceType, _ string) string { return a.Name }),
		CertificateTypes: newMemMaster(func(c *model.CertificateType, id uuid.UUID) { c.ID = id },
			func(c model.CertificateType, _ string) string { return c.Name }),
	}
	changes := f.changes
	f.data = NewMasterData(stores, f.lookup, func(context.Context) error { *changes++; return nil })
	return f
}

func TestValidIMO(t *testing.T) {
	cases := []struct {
		imo  string
		want bool
	}{
		{"9074729", true},
		{"9074728", false},
		{"9176187", true},
		{"9176188", false},
		{"907472", false},
		{"90747290", false},
		{"90747a9", false},
		{"", false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, ValidIMO(tc.imo), tc.imo)
	}
}

func TestVesselRejectsInvalidIMOCheckDigit(t *testing.T) {
	f := newMasterFixture()
	_, err := f.data.Vessels.Create(context.Background(), model.Vessel{Name: "MV Test", IMONumber: "9074728", Status: "active"})
	e := requireAppErr(t, err, "validation_failed")
	assert.Contains(t, e.Fields, "imo_number")

	v, err := f.data.Vessels.Create(context.Background(), model.Vessel{Name: "MV Test", IMONumber: "9074729", Status: "active"})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, v.ID)
	assert.Equal(t, 1, *f.changes, "a successful write invalidates cached options")
}

func TestVesselIMOMustBeUnique(t *testing.T) {
	f := newMasterFixture()
	ctx := context.Background()
	first, err := f.data.Vessels.Create(ctx, model.Vessel{Name: "A", IMONumber: "9074729"})
	require.NoError(t, err)
	_, err = f.data.Vessels.Create(ctx, model.Vessel{Name: "B", IMONumber: "9074729"})
	e := requireAppErr(t, err, "duplicate")
	assert.Equal(t, 409, e.Status)
	assert.Contains(t, e.Fields, "imo_number")

	_, err = f.data.Vessels.Update(ctx, first.ID, model.Vessel{Name: "A renamed", IMONumber: "9074729"})
	assert.NoError(t, err, "a record does not conflict with itself")
}

func TestVesselCompanyTypeCompatibility(t *testing.T) {
	f := newMasterFixture()
	company, principal, both, missing := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	f.lookup.companies[company] = model.CompanyTypeCompany
	f.lookup.companies[principal] = model.CompanyTypePrincipal
	f.lookup.companies[both] = model.CompanyTypeBoth
	ctx := context.Background()

	cases := []struct {
		name      string
		owner     *uuid.UUID
		principal *uuid.UUID
		bad       []string
	}{
		{"company owns, principal operates", &company, &principal, nil},
		{"both type fits either role", &both, &both, nil},
		{"principal cannot own", &principal, nil, []string{"owner_company_id"}},
		{"company cannot be principal", nil, &company, []string{"principal_id"}},
		{"unknown companies", &missing, &missing, []string{"owner_company_id", "principal_id"}},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			imo := []string{"9074729", "9176187", "9234719", "9395044", "9321483"}[i]
			_, err := f.data.Vessels.Create(ctx, model.Vessel{Name: tc.name, IMONumber: imo, OwnerCompanyID: tc.owner, PrincipalID: tc.principal})
			if tc.bad == nil {
				assert.NoError(t, err)
				return
			}
			e := requireAppErr(t, err, "validation_failed")
			for _, field := range tc.bad {
				assert.Contains(t, e.Fields, field)
			}
		})
	}
}

func TestVesselTypeMustExist(t *testing.T) {
	f := newMasterFixture()
	known := uuid.New()
	f.lookup.vesselTypes[known] = true
	unknown := uuid.New()
	_, err := f.data.Vessels.Create(context.Background(), model.Vessel{Name: "X", IMONumber: "9074729", VesselTypeID: &unknown})
	e := requireAppErr(t, err, "validation_failed")
	assert.Contains(t, e.Fields, "vessel_type_id")
	_, err = f.data.Vessels.Create(context.Background(), model.Vessel{Name: "X", IMONumber: "9074729", VesselTypeID: &known})
	assert.NoError(t, err)
}

func TestCompanyLogoMustBeAStoredFile(t *testing.T) {
	f := newMasterFixture()
	logo := uuid.New()
	_, err := f.data.Companies.Create(context.Background(), model.Company{Code: "ACME", Name: "Acme", Type: "both", LogoFileID: &logo})
	e := requireAppErr(t, err, "validation_failed")
	assert.Contains(t, e.Fields, "logo_file_id")

	f.lookup.files[logo] = true
	_, err = f.data.Companies.Create(context.Background(), model.Company{Code: "ACME", Name: "Acme", Type: "both", LogoFileID: &logo})
	assert.NoError(t, err)
}

func TestCompanyCodeUniqueCaseInsensitive(t *testing.T) {
	f := newMasterFixture()
	ctx := context.Background()
	_, err := f.data.Companies.Create(ctx, model.Company{Code: "ACME", Name: "Acme", Type: "both"})
	require.NoError(t, err)
	_, err = f.data.Companies.Create(ctx, model.Company{Code: "acme", Name: "Other", Type: "company"})
	e := requireAppErr(t, err, "duplicate")
	assert.Contains(t, e.Fields, "code")
}

func TestSystemCrewStatusIsProtected(t *testing.T) {
	f := newMasterFixture()
	ctx := context.Background()
	code := "available"
	id := uuid.New()
	f.statuses.items[id] = model.CrewStatus{MasterBase: model.MasterBase{ID: id, IsActive: true}, Code: &code, Name: "Available", Color: "#16A34A", IsSystem: true}

	err := f.data.CrewStatuses.Delete(ctx, id)
	requireAppErr(t, err, "system_record")
	assert.Empty(t, f.statuses.deleted)

	_, err = f.data.CrewStatuses.Update(ctx, id, model.CrewStatus{MasterBase: model.MasterBase{IsActive: false}, Name: "Available", Color: "#16A34A"})
	e := requireAppErr(t, err, "validation_failed")
	assert.Contains(t, e.Fields, "is_active")

	other := "hijacked"
	updated, err := f.data.CrewStatuses.Update(ctx, id, model.CrewStatus{MasterBase: model.MasterBase{IsActive: true}, Code: &other, Name: "Ready", Color: "#22C55E"})
	require.NoError(t, err)
	assert.Equal(t, "available", *updated.Code, "the code of a system status cannot change")
	assert.True(t, updated.IsSystem)
	assert.Equal(t, "Ready", updated.Name)
}

func TestNewCrewStatusIsNeverSystem(t *testing.T) {
	f := newMasterFixture()
	code := "available"
	s, err := f.data.CrewStatuses.Create(context.Background(), model.CrewStatus{Code: &code, IsSystem: true, Name: "Vacation", Color: "#000000"})
	require.NoError(t, err)
	assert.False(t, s.IsSystem)
	assert.Nil(t, s.Code)

	require.NoError(t, f.data.CrewStatuses.Delete(context.Background(), s.ID))
}

func TestRecordInUseCannotBeDeleted(t *testing.T) {
	f := newMasterFixture()
	ctx := context.Background()
	r, err := f.data.Ranks.Create(ctx, model.Rank{Name: "Master", ShortCode: "MST", Department: "deck"})
	require.NoError(t, err)
	f.ranks.usages[r.ID] = []model.Usage{{Label: "crews", Count: 3}, {Label: "contracts", Count: 1}}

	err = f.data.Ranks.Delete(ctx, r.ID)
	e := requireAppErr(t, err, "in_use")
	assert.Equal(t, 409, e.Status)
	assert.Contains(t, e.Message, "3 crews, 1 contracts")
	assert.Equal(t, map[string]string{"crews": "3", "contracts": "1"}, e.Fields)
	assert.Empty(t, f.ranks.deleted)

	delete(f.ranks.usages, r.ID)
	require.NoError(t, f.data.Ranks.Delete(ctx, r.ID))
	assert.Equal(t, []uuid.UUID{r.ID}, f.ranks.deleted)
}

func TestCurrencyCodeIsImmutable(t *testing.T) {
	f := newMasterFixture()
	ctx := context.Background()
	c, err := f.data.Currencies.Create(ctx, model.Currency{Code: "USD", Decimals: 2})
	require.NoError(t, err)
	_, err = f.data.Currencies.Update(ctx, c.ID, model.Currency{Code: "EUR", Decimals: 2})
	e := requireAppErr(t, err, "validation_failed")
	assert.Contains(t, e.Fields, "code")
	_, err = f.data.Currencies.Update(ctx, c.ID, model.Currency{Code: "USD", Name: "US Dollar", Decimals: 2})
	assert.NoError(t, err)
}

func TestDocumentTypeCodeIsPreserved(t *testing.T) {
	f := newMasterFixture()
	ctx := context.Background()
	code := "passport"
	id := uuid.New()
	f.docTypes.items[id] = model.DocumentType{MasterBase: model.MasterBase{ID: id}, Code: &code, Name: "Passport"}
	updated, err := f.data.DocumentTypes.Update(ctx, id, model.DocumentType{Name: "Passport (ICAO)", HasExpiry: true})
	require.NoError(t, err)
	assert.Equal(t, "passport", *updated.Code)
}

func TestGetAndDeleteMissingRecord(t *testing.T) {
	f := newMasterFixture()
	_, err := f.data.Ranks.Get(context.Background(), uuid.New())
	e := requireAppErr(t, err, "not_found")
	assert.Equal(t, "Rank not found.", e.Message)
	requireAppErr(t, f.data.Ranks.Delete(context.Background(), uuid.New()), "not_found")
}

func TestReorderValidation(t *testing.T) {
	f := newMasterFixture()
	ctx := context.Background()
	requireAppErr(t, f.data.Ranks.Reorder(ctx, nil), "validation_failed")
	a := uuid.New()
	requireAppErr(t, f.data.Ranks.Reorder(ctx, []uuid.UUID{a, a}), "validation_failed")
	b := uuid.New()
	require.NoError(t, f.data.Ranks.Reorder(ctx, []uuid.UUID{b, a}))
	assert.Equal(t, []uuid.UUID{b, a}, f.ranks.order)
}
