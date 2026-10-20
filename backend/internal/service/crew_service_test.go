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
)

type memCrews struct {
	byID         map[uuid.UUID]model.Crew
	activeRank   map[uuid.UUID]uuid.UUID
	requirements map[uuid.UUID][]repository.RequirementRow
	held         map[uuid.UUID][]repository.HeldRow
	collisions   int
	createErr    error
}

func newMemCrews() *memCrews {
	return &memCrews{byID: map[uuid.UUID]model.Crew{}, activeRank: map[uuid.UUID]uuid.UUID{},
		requirements: map[uuid.UUID][]repository.RequirementRow{}, held: map[uuid.UUID][]repository.HeldRow{}}
}

func (m *memCrews) Create(_ context.Context, c model.Crew, publicID, status string) (uuid.UUID, error) {
	if m.createErr != nil {
		return uuid.Nil, m.createErr
	}
	if m.collisions > 0 {
		m.collisions--
		return uuid.Nil, &pgconn.PgError{Code: "23505", ConstraintName: "crews_public_id_key"}
	}
	c.ID, c.PublicID, c.StatusCode = uuid.New(), publicID, status
	m.byID[c.ID] = c
	return c.ID, nil
}

func (m *memCrews) Update(_ context.Context, id uuid.UUID, c model.Crew) error {
	cur := m.byID[id]
	c.ID, c.PublicID, c.StatusCode = cur.ID, cur.PublicID, cur.StatusCode
	m.byID[id] = c
	return nil
}

func (m *memCrews) GetByPublicID(_ context.Context, pid string) (model.Crew, error) {
	for _, c := range m.byID {
		if c.PublicID == pid {
			return c, nil
		}
	}
	return model.Crew{}, repository.ErrNotFound
}

func (m *memCrews) GetByID(_ context.Context, id uuid.UUID) (model.Crew, error) {
	c, ok := m.byID[id]
	if !ok {
		return c, repository.ErrNotFound
	}
	return c, nil
}

func (m *memCrews) SoftDelete(_ context.Context, id uuid.UUID) error {
	delete(m.byID, id)
	return nil
}

func (m *memCrews) SetStatus(_ context.Context, id uuid.UUID, code, reason string) error {
	c := m.byID[id]
	c.StatusCode, c.StatusReason = code, reason
	m.byID[id] = c
	return nil
}

func (m *memCrews) HasActiveContract(_ context.Context, id uuid.UUID) (bool, error) {
	_, ok := m.activeRank[id]
	return ok, nil
}

func (m *memCrews) ActiveContractRank(_ context.Context, id uuid.UUID) (*uuid.UUID, error) {
	r, ok := m.activeRank[id]
	if !ok {
		return nil, nil
	}
	return &r, nil
}

func (m *memCrews) Requirements(_ context.Context, rankID uuid.UUID) ([]repository.RequirementRow, error) {
	return m.requirements[rankID], nil
}

func (m *memCrews) HeldItems(_ context.Context, id uuid.UUID) ([]repository.HeldRow, error) {
	return m.held[id], nil
}

func (m *memCrews) ListCrews(context.Context, model.CrewFilter, int, int) (repository.Page[model.CrewRow], []model.CrewGroup, error) {
	return repository.Page[model.CrewRow]{}, nil, nil
}

func (m *memCrews) StreamCrews(context.Context, model.CrewFilter, int, func(model.CrewRow) error) error {
	return nil
}

type memRecords struct {
	CrewRecordStore
	experiences map[uuid.UUID]model.Experience
	typeExpiry  map[uuid.UUID]bool
	docs        map[uuid.UUID]model.CrewDocument
}

func (m *memRecords) ListExperiences(_ context.Context, crewID uuid.UUID) ([]model.Experience, error) {
	var out []model.Experience
	for _, x := range m.experiences {
		if x.CrewID == crewID {
			out = append(out, x)
		}
	}
	return out, nil
}

func (m *memRecords) GetExperience(_ context.Context, _, id uuid.UUID) (model.Experience, error) {
	x, ok := m.experiences[id]
	if !ok {
		return x, repository.ErrNotFound
	}
	return x, nil
}

func (m *memRecords) DeleteExperience(_ context.Context, _, id uuid.UUID) error {
	delete(m.experiences, id)
	return nil
}

func (m *memRecords) TypeHasExpiry(_ context.Context, _ repository.DocumentKind, id uuid.UUID) (bool, error) {
	has, ok := m.typeExpiry[id]
	if !ok {
		return false, repository.ErrNotFound
	}
	return has, nil
}

func (m *memRecords) SaveDocument(_ context.Context, _ repository.DocumentKind, d model.CrewDocument) (uuid.UUID, error) {
	d.ID = uuid.New()
	d.HasExpiry = m.typeExpiry[d.TypeID]
	m.docs[d.ID] = d
	return d.ID, nil
}

func (m *memRecords) GetDocument(_ context.Context, _ repository.DocumentKind, _, id uuid.UUID) (model.CrewDocument, error) {
	return m.docs[id], nil
}

type crewFixture struct {
	svc     *CrewService
	crews   *memCrews
	records *memRecords
}

func newCrewFixture(t *testing.T) crewFixture {
	t.Helper()
	crews := newMemCrews()
	records := &memRecords{experiences: map[uuid.UUID]model.Experience{}, typeExpiry: map[uuid.UUID]bool{}, docs: map[uuid.UUID]model.CrewDocument{}}
	settings := staticSettings{
		"general": {"timezone": "Asia/Jakarta"},
		"alerts":  {"expiry_threshold_1": 30, "expiry_threshold_2": 60, "expiry_threshold_3": 90},
	}
	now := func() time.Time { return time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC) }
	return crewFixture{svc: NewCrewService(crews, records, settings, now), crews: crews, records: records}
}

func TestCreateCrewStartsAvailableWithValidPublicID(t *testing.T) {
	f := newCrewFixture(t)
	c, err := f.svc.Create(context.Background(), model.Crew{FullName: "Budi Santoso"})
	require.NoError(t, err)
	assert.Equal(t, StatusAvailable, c.StatusCode)
	assert.Regexp(t, `^[2-9A-HJKMNP-Z]{9}$`, c.PublicID)
}

func TestCreateCrewRetriesOnPublicIDCollision(t *testing.T) {
	f := newCrewFixture(t)
	f.crews.collisions = 4
	_, err := f.svc.Create(context.Background(), model.Crew{FullName: "A"})
	require.NoError(t, err)

	f.crews.collisions = 5
	_, err = f.svc.Create(context.Background(), model.Crew{FullName: "B"})
	assert.Error(t, err, "gives up after five attempts")
}

func TestCreateCrewMapsUnknownRankToValidationError(t *testing.T) {
	f := newCrewFixture(t)
	f.crews.createErr = &pgconn.PgError{Code: "23503", ConstraintName: "crews_applied_rank_id_fkey"}
	_, err := f.svc.Create(context.Background(), model.Crew{FullName: "A"})
	e := requireAppErr(t, err, "validation_failed")
	assert.Contains(t, e.Fields, "applied_rank_id")
}

func TestGetRejectsMalformedPublicID(t *testing.T) {
	f := newCrewFixture(t)
	_, err := f.svc.Get(context.Background(), "not-an-id")
	requireAppErr(t, err, "not_found")
}

func TestDeleteAndStatusChangeBlockedWhileOnBoard(t *testing.T) {
	f := newCrewFixture(t)
	ctx := context.Background()
	c, err := f.svc.Create(ctx, model.Crew{FullName: "A"})
	require.NoError(t, err)
	f.crews.activeRank[c.ID] = uuid.New()

	requireAppErr(t, f.svc.Delete(ctx, c.PublicID), "crew_on_board")
	_, err = f.svc.ChangeStatus(ctx, c.PublicID, ActionMedicalLeave, "")
	requireAppErr(t, err, "crew_on_board")

	delete(f.crews.activeRank, c.ID)
	updated, err := f.svc.ChangeStatus(ctx, c.PublicID, ActionBlacklist, "Fake certificate")
	require.NoError(t, err)
	assert.Equal(t, StatusBlacklisted, updated.StatusCode)
	assert.Equal(t, "Fake certificate", updated.StatusReason)
	require.NoError(t, f.svc.Delete(ctx, c.PublicID))
}

func TestComplianceUsesActiveContractRankBeforeAppliedRank(t *testing.T) {
	f := newCrewFixture(t)
	ctx := context.Background()
	applied, contractRank, passport := uuid.New(), uuid.New(), uuid.New()
	c, err := f.svc.Create(ctx, model.Crew{FullName: "A", AppliedRankID: &applied})
	require.NoError(t, err)
	f.crews.requirements[applied] = []repository.RequirementRow{{Kind: "document", TypeID: passport, TypeName: "Passport", HasExpiry: true}}
	f.crews.requirements[contractRank] = nil

	res, err := f.svc.Compliance(ctx, c.PublicID, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, applied, *res.RankID)
	assert.False(t, res.Compliant)
	assert.Equal(t, time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC), res.CheckDate, "today in WIB")

	f.crews.activeRank[c.ID] = contractRank
	res, err = f.svc.Compliance(ctx, c.PublicID, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, contractRank, *res.RankID)
	assert.True(t, res.Compliant)
}

func TestComplianceWithoutRankIsNotCompliant(t *testing.T) {
	f := newCrewFixture(t)
	c, err := f.svc.Create(context.Background(), model.Crew{FullName: "A"})
	require.NoError(t, err)
	res, err := f.svc.Compliance(context.Background(), c.PublicID, nil, nil)
	require.NoError(t, err)
	assert.Nil(t, res.RankID)
	assert.False(t, res.Compliant)
}

func TestSaveDocumentRequiresExpiryForExpiringTypes(t *testing.T) {
	f := newCrewFixture(t)
	ctx := context.Background()
	c, err := f.svc.Create(ctx, model.Crew{FullName: "A"})
	require.NoError(t, err)
	passport, ktp := uuid.New(), uuid.New()
	f.records.typeExpiry[passport] = true
	f.records.typeExpiry[ktp] = false

	_, err = f.svc.SaveDocument(ctx, repository.TravelDocuments, c.PublicID, model.CrewDocument{TypeID: passport, Number: "X1"})
	e := requireAppErr(t, err, "validation_failed")
	assert.Contains(t, e.Fields, "expired_date")

	exp := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	view, err := f.svc.SaveDocument(ctx, repository.TravelDocuments, c.PublicID, model.CrewDocument{TypeID: passport, Number: "X1", ExpiredDate: &exp})
	require.NoError(t, err)
	assert.Equal(t, BadgeRed, view.Badge)
	assert.Equal(t, 21, *view.DaysLeft)

	view, err = f.svc.SaveDocument(ctx, repository.TravelDocuments, c.PublicID, model.CrewDocument{TypeID: ktp, Number: "317", ExpiredDate: &exp})
	require.NoError(t, err)
	assert.Nil(t, view.ExpiredDate, "expiry is dropped for types without expiry")
	assert.Equal(t, BadgeGreen, view.Badge)

	_, err = f.svc.SaveDocument(ctx, repository.TravelDocuments, c.PublicID, model.CrewDocument{TypeID: uuid.New(), Number: "?"})
	e = requireAppErr(t, err, "validation_failed")
	assert.Contains(t, e.Fields, "type_id")
}

func TestContractExperienceIsReadOnly(t *testing.T) {
	f := newCrewFixture(t)
	ctx := context.Background()
	c, err := f.svc.Create(ctx, model.Crew{FullName: "A"})
	require.NoError(t, err)
	id := uuid.New()
	f.records.experiences[id] = model.Experience{ID: id, CrewID: c.ID, Source: "contract"}

	requireAppErr(t, f.svc.DeleteExperience(ctx, c.PublicID, id), "experience_from_contract")
	_, err = f.svc.SaveExperience(ctx, c.PublicID, model.Experience{ID: id})
	requireAppErr(t, err, "experience_from_contract")
	requireAppErr(t, f.svc.DeleteExperience(ctx, c.PublicID, uuid.New()), "not_found")
}

func TestSeaServiceTotalsPerRank(t *testing.T) {
	f := newCrewFixture(t)
	ctx := context.Background()
	c, err := f.svc.Create(ctx, model.Crew{FullName: "A"})
	require.NoError(t, err)
	master := uuid.New()
	off := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	f.records.experiences[uuid.New()] = model.Experience{CrewID: c.ID, RankID: master, RankName: "Master", SignOnDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), SignOffDate: &off}
	f.records.experiences[uuid.New()] = model.Experience{CrewID: c.ID, RankID: master, RankName: "Master", SignOnDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}

	res, err := f.svc.SeaService(ctx, c.PublicID)
	require.NoError(t, err)
	require.Len(t, res.ByRank, 1)
	assert.Equal(t, "Master", res.ByRank[0].RankName)
	assert.Equal(t, 51, res.Total.TotalDays, "31 days plus 1–20 Oct open row")
}

func TestLoadDisplaySettingsFallsBackToDefaults(t *testing.T) {
	loc, th, err := LoadDisplaySettings(context.Background(), staticSettings{"general": {}, "alerts": {}})
	require.NoError(t, err)
	assert.Equal(t, "Asia/Jakarta", loc.String())
	assert.Equal(t, DefaultThresholds, th)
}
