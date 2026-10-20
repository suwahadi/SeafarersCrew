package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

type memPlans struct {
	plans          map[uuid.UUID]model.ReliefPlan
	contracts      map[uuid.UUID]model.Contract
	suggestionArgs []time.Time
	completed      map[uuid.UUID]uuid.UUID
}

func (m *memPlans) Get(_ context.Context, id uuid.UUID) (model.ReliefPlan, error) {
	p, ok := m.plans[id]
	if !ok {
		return p, repository.ErrNotFound
	}
	k := m.contracts[p.ContractID]
	p.ContractPublicID, p.ContractDuration, p.RankID = k.PublicID, k.DurationMonths, k.RankID
	return p, nil
}

func (m *memPlans) List(context.Context, model.ReliefPlanFilter, int, int) (repository.Page[model.ReliefPlan], error) {
	return repository.Page[model.ReliefPlan]{}, nil
}

func (m *memPlans) Create(_ context.Context, p model.ReliefPlan) (uuid.UUID, error) {
	p.ID = uuid.New()
	m.plans[p.ID] = p
	return p.ID, nil
}

func (m *memPlans) Update(_ context.Context, p model.ReliefPlan) error {
	m.plans[p.ID] = p
	return nil
}

func (m *memPlans) Delete(_ context.Context, id uuid.UUID) error {
	if m.plans[id].Status == model.ReliefCompleted {
		return repository.ErrStateChanged
	}
	delete(m.plans, id)
	return nil
}

func (m *memPlans) OpenWindows(_ context.Context, relieverID, exclude uuid.UUID) ([]model.ReliefWindow, error) {
	var out []model.ReliefWindow
	for _, p := range m.plans {
		if p.RelieverCrewID == relieverID && p.ID != exclude && (p.Status == model.ReliefPlanned || p.Status == model.ReliefConfirmed) {
			out = append(out, model.ReliefWindow{PlanID: p.ID, ReliefDate: p.PlannedReliefDate, DurationMonths: m.contracts[p.ContractID].DurationMonths})
		}
	}
	return out, nil
}

func (m *memPlans) Suggestions(_ context.Context, _, _ uuid.UUID, date, end time.Time, _ int) ([]model.ReliefCandidate, error) {
	m.suggestionArgs = []time.Time{date, end}
	return []model.ReliefCandidate{{PublicID: "SUGGEST22"}}, nil
}

func (m *memPlans) TimelineVessels(context.Context, []uuid.UUID) ([]model.TimelineVessel, error) {
	return nil, nil
}

func (m *memPlans) TimelineBars(context.Context, time.Time, time.Time, []uuid.UUID) ([]model.TimelineBar, error) {
	return nil, nil
}

func (m *memPlans) OpenPlansForContracts(context.Context, []uuid.UUID) ([]model.ReliefPlan, error) {
	return nil, nil
}

func (m *memPlans) Complete(_ context.Context, id uuid.UUID, _ func() (string, error), _ int) (uuid.UUID, error) {
	p := m.plans[id]
	if p.Status == model.ReliefCancelled {
		return uuid.Nil, repository.ErrPlanCancelled
	}
	if existing, ok := m.completed[id]; ok {
		return existing, nil
	}
	newID := uuid.New()
	m.completed[id] = newID
	m.contracts[newID] = model.Contract{ID: newID, PublicID: "NEWCONTR2", Status: model.ContractDraft, CrewID: p.RelieverCrewID}
	p.Status = model.ReliefCompleted
	m.plans[id] = p
	return newID, nil
}

type memContractReader struct{ m *memPlans }

func (r memContractReader) Get(_ context.Context, pid string) (model.Contract, error) {
	for _, k := range r.m.contracts {
		if k.PublicID == pid {
			return k, nil
		}
	}
	return model.Contract{}, repository.ErrNotFound
}

func (r memContractReader) GetByID(_ context.Context, id uuid.UUID) (model.Contract, error) {
	return r.m.contracts[id], nil
}

type fakeCompliance struct {
	byDate map[time.Time]ComplianceResult
}

func (f fakeCompliance) Compliance(_ context.Context, _ string, rankID *uuid.UUID, d *time.Time) (CrewCompliance, error) {
	return CrewCompliance{RankID: rankID, ComplianceResult: f.byDate[*d]}, nil
}

type rotationFixture struct {
	svc      *RotationService
	plans    *memPlans
	contract model.Contract
	reliever uuid.UUID
	compl    fakeCompliance
}

func newRotationFixture(t *testing.T) rotationFixture {
	t.Helper()
	contract := model.Contract{ID: uuid.New(), PublicID: "CONTRACT2", CrewID: uuid.New(), RankID: uuid.New(), Status: model.ContractActive,
		SignOnDate: date(2026, 5, 1), DurationMonths: 6, PlannedSignOffDate: date(2026, 11, 1)}
	plans := &memPlans{plans: map[uuid.UUID]model.ReliefPlan{}, contracts: map[uuid.UUID]model.Contract{contract.ID: contract}, completed: map[uuid.UUID]uuid.UUID{}}
	reliever := uuid.New()
	crews := fakeCrewFinder{
		"RELIEVER2": {ID: reliever, PublicID: "RELIEVER2", StatusCode: StatusAvailable},
		"ONLEAVE22": {ID: uuid.New(), PublicID: "ONLEAVE22", StatusCode: StatusMedicalLeave},
		"BLACKLST2": {ID: uuid.New(), PublicID: "BLACKLST2", StatusCode: StatusBlacklisted},
		"RELIEVED2": {ID: contract.CrewID, PublicID: "RELIEVED2", StatusCode: StatusOnBoard},
	}
	compl := fakeCompliance{byDate: map[time.Time]ComplianceResult{}}
	settings := staticSettings{"general": {"timezone": "Asia/Jakarta"}, "alerts": {}}
	reader := memContractReader{m: plans}
	svc := NewRotationService(plans, reader, reader, crews, compl, settings, func() time.Time { return time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC) })
	return rotationFixture{svc: svc, plans: plans, contract: contract, reliever: reliever, compl: compl}
}

func (f rotationFixture) input(reliever string, d time.Time) ReliefInput {
	return ReliefInput{ContractPublicID: f.contract.PublicID, RelieverPublicID: reliever, PlannedReliefDate: d, JoiningPort: "Batam"}
}

func TestCreateReliefPlanWithComplianceWarnings(t *testing.T) {
	f := newRotationFixture(t)
	passport := uuid.New()
	exp := date(2027, 2, 1)
	f.compl.byDate[date(2026, 11, 1)] = ComplianceResult{Items: []ComplianceItem{{TypeID: passport, Status: ItemValid}}}
	f.compl.byDate[date(2027, 5, 1)] = ComplianceResult{Items: []ComplianceItem{{Kind: "document", TypeID: passport, TypeName: "Passport", Status: ItemExpired, ExpiredDate: &exp}}}

	res, err := f.svc.Create(context.Background(), f.input("RELIEVER2", date(2026, 11, 1)))
	require.NoError(t, err)
	assert.Equal(t, model.ReliefPlanned, res.Plan.Status)
	assert.Equal(t, f.reliever, res.Plan.RelieverCrewID)
	require.Len(t, res.Warnings, 1)
	assert.Equal(t, WarnExpiresBeforeEnd, res.Warnings[0].Reason)
}

func TestReliefPlanRejections(t *testing.T) {
	cases := []struct {
		name     string
		reliever string
		date     time.Time
		mutate   func(f *rotationFixture)
		code     string
	}{
		{"reliever on medical leave", "ONLEAVE22", date(2026, 11, 1), nil, "reliever_not_available"},
		{"blacklisted reliever", "BLACKLST2", date(2026, 11, 1), nil, "reliever_not_available"},
		{"reliever is the relieved crew", "RELIEVED2", date(2026, 11, 1), nil, "validation_failed"},
		{"unknown reliever", "NOBODY222", date(2026, 11, 1), nil, "validation_failed"},
		{"relief before sign-on", "RELIEVER2", date(2026, 4, 1), nil, "validation_failed"},
		{"contract no longer active", "RELIEVER2", date(2026, 11, 1), func(f *rotationFixture) {
			k := f.plans.contracts[f.contract.ID]
			k.Status = model.ContractCompleted
			f.plans.contracts[f.contract.ID] = k
		}, "contract_not_active"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newRotationFixture(t)
			if tc.mutate != nil {
				tc.mutate(&f)
			}
			_, err := f.svc.Create(context.Background(), f.input(tc.reliever, tc.date))
			requireAppErr(t, err, tc.code)
		})
	}
}

func TestRelieverCannotBePlannedOnOverlappingReliefs(t *testing.T) {
	f := newRotationFixture(t)
	ctx := context.Background()
	first, err := f.svc.Create(ctx, f.input("RELIEVER2", date(2026, 11, 1)))
	require.NoError(t, err)

	_, err = f.svc.Create(ctx, f.input("RELIEVER2", date(2027, 3, 1)))
	requireAppErr(t, err, "reliever_overlap")

	_, err = f.svc.Create(ctx, f.input("RELIEVER2", date(2027, 5, 2)))
	assert.NoError(t, err, "starts the day after the six-month window ends")

	moved := f.input("RELIEVER2", date(2026, 10, 25))
	_, err = f.svc.Update(ctx, first.Plan.ID, moved)
	assert.NoError(t, err, "a plan never conflicts with itself")

	cancel := moved
	cancel.Status = model.ReliefCancelled
	res, err := f.svc.Update(ctx, first.Plan.ID, cancel)
	require.NoError(t, err)
	assert.Equal(t, model.ReliefCancelled, res.Plan.Status)

	_, err = f.svc.Update(ctx, first.Plan.ID, moved)
	requireAppErr(t, err, "relief_plan_closed")
	_, err = f.svc.Complete(ctx, first.Plan.ID)
	requireAppErr(t, err, "relief_plan_closed")
}

func TestCompleteReliefIsIdempotent(t *testing.T) {
	f := newRotationFixture(t)
	ctx := context.Background()
	res, err := f.svc.Create(ctx, f.input("RELIEVER2", date(2026, 11, 1)))
	require.NoError(t, err)

	k1, err := f.svc.Complete(ctx, res.Plan.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ContractDraft, k1.Status)
	assert.Equal(t, f.reliever, k1.CrewID)
	k2, err := f.svc.Complete(ctx, res.Plan.ID)
	require.NoError(t, err)
	assert.Equal(t, k1.ID, k2.ID)

	requireAppErr(t, f.svc.Delete(ctx, res.Plan.ID), "relief_plan_closed")
}

func TestSuggestionsDefaultToPlannedSignOff(t *testing.T) {
	f := newRotationFixture(t)
	list, d, err := f.svc.Suggestions(context.Background(), "CONTRACT2", nil)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, date(2026, 11, 1), d)
	assert.Equal(t, []time.Time{date(2026, 11, 1), date(2027, 5, 1)}, f.plans.suggestionArgs)

	custom := date(2026, 12, 15)
	_, d, err = f.svc.Suggestions(context.Background(), "CONTRACT2", &custom)
	require.NoError(t, err)
	assert.Equal(t, custom, d)
}

func TestTimelineValidatesRange(t *testing.T) {
	f := newRotationFixture(t)
	from, to := date(2026, 10, 1), date(2026, 9, 1)
	_, err := f.svc.Timeline(context.Background(), &from, &to, nil)
	requireAppErr(t, err, "validation_failed")

	tl, err := f.svc.Timeline(context.Background(), nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, date(2026, 10, 20), tl.Today)
	assert.Equal(t, date(2026, 9, 20), tl.From)
	assert.Equal(t, date(2027, 4, 18), tl.To)
}
