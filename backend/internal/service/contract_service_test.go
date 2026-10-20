package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type memContracts struct {
	byID        map[uuid.UUID]model.Contract
	crewStatus  map[uuid.UUID]string
	experiences map[uuid.UUID]model.Experience
	collisions  int
	activateErr error
}

func newMemContracts() *memContracts {
	return &memContracts{byID: map[uuid.UUID]model.Contract{}, crewStatus: map[uuid.UUID]string{}, experiences: map[uuid.UUID]model.Experience{}}
}

func (m *memContracts) Create(_ context.Context, k model.Contract, pid string) (uuid.UUID, error) {
	if m.collisions > 0 {
		m.collisions--
		return uuid.Nil, &pgconn.PgError{Code: "23505", ConstraintName: "contracts_public_id_key"}
	}
	k.ID, k.PublicID, k.Status = uuid.New(), pid, model.ContractDraft
	m.byID[k.ID] = k
	return k.ID, nil
}

func (m *memContracts) Update(_ context.Context, k model.Contract, status string) error {
	cur := m.byID[k.ID]
	if cur.Status != status {
		return repository.ErrStateChanged
	}
	k.PublicID, k.Status = cur.PublicID, cur.Status
	m.byID[k.ID] = k
	return nil
}

func (m *memContracts) fill(k model.Contract) model.Contract {
	k.CrewStatusCode = m.crewStatus[k.CrewID]
	return k
}

func (m *memContracts) GetByPublicID(_ context.Context, pid string) (model.Contract, error) {
	for _, k := range m.byID {
		if k.PublicID == pid {
			return m.fill(k), nil
		}
	}
	return model.Contract{}, repository.ErrNotFound
}

func (m *memContracts) GetByID(_ context.Context, id uuid.UUID) (model.Contract, error) {
	k, ok := m.byID[id]
	if !ok {
		return k, repository.ErrNotFound
	}
	return m.fill(k), nil
}

func (m *memContracts) ActiveForCrew(_ context.Context, crewID uuid.UUID) (model.Contract, error) {
	for _, k := range m.byID {
		if k.CrewID == crewID && k.Status == model.ContractActive {
			return k, nil
		}
	}
	return model.Contract{}, repository.ErrNotFound
}

func (m *memContracts) List(context.Context, model.ContractFilter, int, int) (repository.Page[model.Contract], error) {
	return repository.Page[model.Contract]{}, nil
}

func (m *memContracts) SoftDelete(_ context.Context, id uuid.UUID) error {
	delete(m.byID, id)
	return nil
}

func (m *memContracts) Activate(_ context.Context, id, crewID uuid.UUID) error {
	if m.activateErr != nil {
		return m.activateErr
	}
	k := m.byID[id]
	k.Status = model.ContractActive
	m.byID[id] = k
	m.crewStatus[crewID] = StatusOnBoard
	return nil
}

func (m *memContracts) Finish(_ context.Context, id, crewID uuid.UUID, status string, actual time.Time, reason string, x model.Experience) error {
	k := m.byID[id]
	if k.Status != model.ContractActive {
		return repository.ErrStateChanged
	}
	k.Status, k.ActualSignOffDate, k.TerminationReason = status, &actual, reason
	m.byID[id] = k
	if _, exists := m.experiences[id]; !exists {
		m.experiences[id] = x
	}
	m.crewStatus[crewID] = StatusSignOff
	return nil
}

type fakeContractLookups struct {
	companies map[uuid.UUID]string
	vessels   map[uuid.UUID]bool
	ranks     map[uuid.UUID]bool
	allowance map[uuid.UUID]string
}

func (f fakeContractLookups) CompanyType(_ context.Context, id uuid.UUID) (string, error) {
	t, ok := f.companies[id]
	if !ok {
		return "", repository.ErrNotFound
	}
	return t, nil
}

func (f fakeContractLookups) VesselActive(_ context.Context, id uuid.UUID) (bool, error) {
	return f.vessels[id], nil
}

func (f fakeContractLookups) RankActive(_ context.Context, id uuid.UUID) (bool, error) {
	return f.ranks[id], nil
}

func (f fakeContractLookups) CurrencyDecimals(_ context.Context, code string) (int, error) {
	switch code {
	case "USD":
		return 2, nil
	case "IDR":
		return 0, nil
	}
	return 0, repository.ErrNotFound
}

func (f fakeContractLookups) AllowanceType(_ context.Context, id uuid.UUID) (string, string, error) {
	basis, ok := f.allowance[id]
	if !ok {
		return "", "", repository.ErrNotFound
	}
	return "Allowance", basis, nil
}

type fakeCrewFinder map[string]model.Crew

func (f fakeCrewFinder) Get(_ context.Context, pid string) (model.Crew, error) {
	c, ok := f[pid]
	if !ok {
		return c, apperr.NotFound("Crew")
	}
	return c, nil
}

type fakeNextRelief map[uuid.UUID]time.Time

func (f fakeNextRelief) NextReliefDate(_ context.Context, id uuid.UUID) (*time.Time, error) {
	d, ok := f[id]
	if !ok {
		return nil, nil
	}
	return &d, nil
}

type contractFixture struct {
	svc                                     *ContractService
	store                                   *memContracts
	crews                                   fakeCrewFinder
	reliefs                                 fakeNextRelief
	crewID                                  uuid.UUID
	vessel, rank, principal, employer, both uuid.UUID
	overtime, daily, percent                uuid.UUID
}

func newContractFixture(t *testing.T) contractFixture {
	t.Helper()
	f := contractFixture{store: newMemContracts(), crewID: uuid.New(), vessel: uuid.New(), rank: uuid.New(), principal: uuid.New(),
		employer: uuid.New(), both: uuid.New(), overtime: uuid.New(), daily: uuid.New(), percent: uuid.New(), reliefs: fakeNextRelief{}}
	f.crews = fakeCrewFinder{"CREWAAAA2": {ID: f.crewID, PublicID: "CREWAAAA2", StatusCode: StatusAvailable}}
	f.store.crewStatus[f.crewID] = StatusAvailable
	lookups := fakeContractLookups{
		companies: map[uuid.UUID]string{f.principal: "principal", f.employer: "company", f.both: "both"},
		vessels:   map[uuid.UUID]bool{f.vessel: true},
		ranks:     map[uuid.UUID]bool{f.rank: true},
		allowance: map[uuid.UUID]string{f.overtime: model.BasisFixedMonth, f.daily: model.BasisFixedDay, f.percent: model.BasisPercentBasic},
	}
	settings := staticSettings{"general": {"timezone": "Asia/Jakarta"}, "alerts": {}}
	now := func() time.Time { return time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC) }
	f.svc = NewContractService(f.store, lookups, f.crews, f.reliefs, settings, now)
	return f
}

func (f contractFixture) input() model.Contract {
	return model.Contract{CrewPublicID: "CREWAAAA2", VesselID: f.vessel, RankID: f.rank, PrincipalID: f.principal,
		EmployerCompanyID: f.employer, SignOnDate: date(2026, 11, 1), DurationMonths: 6, CurrencyCode: "USD",
		BasicSalary: decimal.RequireFromString("4800"),
		Allowances: []model.ContractAllowance{
			{AllowanceTypeID: f.overtime, Value: decimal.RequireFromString("500")},
			{AllowanceTypeID: f.daily, Value: decimal.RequireFromString("20")},
			{AllowanceTypeID: f.percent, Value: decimal.RequireFromString("10")},
		}}
}

func TestCreateContractComputesPlannedSignOffAndAllowances(t *testing.T) {
	f := newContractFixture(t)
	f.store.collisions = 2
	k, err := f.svc.Create(context.Background(), f.input())
	require.NoError(t, err)
	assert.Equal(t, model.ContractDraft, k.Status)
	assert.Equal(t, date(2027, 5, 1), k.PlannedSignOffDate)
	require.Len(t, k.Allowances, 3)
	assert.True(t, decimal.RequireFromString("500").Equal(k.Allowances[0].MonthlyAmount))
	assert.True(t, decimal.RequireFromString("600").Equal(k.Allowances[1].MonthlyAmount))
	assert.True(t, decimal.RequireFromString("480").Equal(k.Allowances[2].MonthlyAmount))
	assert.True(t, decimal.RequireFromString("6380").Equal(TotalMonthlyPay(k.BasicSalary, k.Allowances)))
}

func TestCreateContractValidatesReferences(t *testing.T) {
	f := newContractFixture(t)
	in := f.input()
	in.CrewPublicID = "MISSING22"
	in.VesselID = uuid.New()
	in.PrincipalID = f.employer
	in.EmployerCompanyID = f.principal
	in.CurrencyCode = "XXX"
	in.PlannedSignOffDate = date(2026, 10, 1)
	in.Allowances = append(in.Allowances, model.ContractAllowance{AllowanceTypeID: uuid.New()},
		model.ContractAllowance{AllowanceTypeID: f.percent, Value: decimal.RequireFromString("150")})
	_, err := f.svc.Create(context.Background(), in)
	e := requireAppErr(t, err, "validation_failed")
	for _, field := range []string{"crew_id", "vessel_id", "principal_id", "employer_company_id", "currency_code",
		"planned_sign_off_date", "allowances.3.allowance_type_id", "allowances.4.value"} {
		assert.Contains(t, e.Fields, field)
	}
}

func TestCreateContractAcceptsCompaniesOfTypeBoth(t *testing.T) {
	f := newContractFixture(t)
	in := f.input()
	in.PrincipalID, in.EmployerCompanyID = f.both, f.both
	_, err := f.svc.Create(context.Background(), in)
	assert.NoError(t, err)
}

func TestIDRContractRoundsToWholeRupiah(t *testing.T) {
	f := newContractFixture(t)
	in := f.input()
	in.CurrencyCode = "IDR"
	in.BasicSalary = decimal.RequireFromString("12345678.6")
	in.Allowances = []model.ContractAllowance{{AllowanceTypeID: f.percent, Value: decimal.RequireFromString("7.5")}}
	k, err := f.svc.Create(context.Background(), in)
	require.NoError(t, err)
	assert.Equal(t, "12345679", k.BasicSalary.String())
	assert.Equal(t, "925926", k.Allowances[0].MonthlyAmount.String())
}

func TestContractLifecycle(t *testing.T) {
	f := newContractFixture(t)
	ctx := context.Background()
	k, err := f.svc.Create(ctx, f.input())
	require.NoError(t, err)

	_, err = f.svc.Complete(ctx, k.PublicID, date(2027, 5, 1))
	requireAppErr(t, err, "contract_not_active")

	k, err = f.svc.Activate(ctx, k.PublicID)
	require.NoError(t, err)
	assert.Equal(t, model.ContractActive, k.Status)
	assert.Equal(t, StatusOnBoard, f.store.crewStatus[f.crewID], "activation puts the crew On Board")

	_, err = f.svc.Activate(ctx, k.PublicID)
	requireAppErr(t, err, "contract_not_draft")
	requireAppErr(t, f.svc.Delete(ctx, k.PublicID), "contract_not_draft")

	_, err = f.svc.Terminate(ctx, k.PublicID, date(2027, 1, 15), "  ")
	requireAppErr(t, err, "validation_failed")

	k, err = f.svc.Terminate(ctx, k.PublicID, date(2027, 1, 15), "Medical repatriation")
	require.NoError(t, err)
	assert.Equal(t, model.ContractTerminated, k.Status)
	assert.Equal(t, StatusSignOff, f.store.crewStatus[f.crewID])
	require.Len(t, f.store.experiences, 1)
	assert.Equal(t, "Medical repatriation", f.store.experiences[k.ID].Remarks)

	_, err = f.svc.Complete(ctx, k.PublicID, date(2027, 2, 1))
	requireAppErr(t, err, "contract_not_active")
	assert.Len(t, f.store.experiences, 1, "finishing twice never adds a second Experience")

	_, err = f.svc.Update(ctx, k.PublicID, f.input())
	requireAppErr(t, err, "contract_closed")
}

func TestSecondActiveContractIsRejected(t *testing.T) {
	f := newContractFixture(t)
	ctx := context.Background()
	first, err := f.svc.Create(ctx, f.input())
	require.NoError(t, err)
	second, err := f.svc.Create(ctx, f.input())
	require.NoError(t, err)
	_, err = f.svc.Activate(ctx, first.PublicID)
	require.NoError(t, err)

	f.store.crewStatus[f.crewID] = StatusAvailable
	_, err = f.svc.Activate(ctx, second.PublicID)
	requireAppErr(t, err, "crew_has_active_contract")

	f.store.activateErr = repository.ErrCrewHasActiveContract
	k := f.store.byID[first.ID]
	k.Status = model.ContractCompleted
	f.store.byID[first.ID] = k
	_, err = f.svc.Activate(ctx, second.PublicID)
	requireAppErr(t, err, "crew_has_active_contract")
}

func TestActivationRequiresAvailableCrew(t *testing.T) {
	f := newContractFixture(t)
	k, err := f.svc.Create(context.Background(), f.input())
	require.NoError(t, err)
	f.store.crewStatus[f.crewID] = StatusMedicalLeave
	_, err = f.svc.Activate(context.Background(), k.PublicID)
	requireAppErr(t, err, "crew_not_available")
}

func TestActiveContractAllowsOnlyLimitedEdits(t *testing.T) {
	f := newContractFixture(t)
	ctx := context.Background()
	k, err := f.svc.Create(ctx, f.input())
	require.NoError(t, err)
	_, err = f.svc.Activate(ctx, k.PublicID)
	require.NoError(t, err)

	edit := f.input()
	edit.PlannedSignOffDate = date(2027, 6, 15)
	edit.SignOffPort, edit.Notes = "Singapore", "Extended one month"
	updated, err := f.svc.Update(ctx, k.PublicID, edit)
	require.NoError(t, err)
	assert.Equal(t, date(2027, 6, 15), updated.PlannedSignOffDate)
	assert.Equal(t, "Singapore", updated.SignOffPort)

	edit.BasicSalary = decimal.RequireFromString("5000")
	_, err = f.svc.Update(ctx, k.PublicID, edit)
	e := requireAppErr(t, err, "contract_locked")
	assert.Contains(t, e.Fields, "basic_salary")
}

func TestDraftContractIsFreelyEditableAndDeletable(t *testing.T) {
	f := newContractFixture(t)
	ctx := context.Background()
	k, err := f.svc.Create(ctx, f.input())
	require.NoError(t, err)
	edit := f.input()
	edit.BasicSalary = decimal.RequireFromString("5200")
	edit.DurationMonths = 9
	updated, err := f.svc.Update(ctx, k.PublicID, edit)
	require.NoError(t, err)
	assert.Equal(t, date(2027, 8, 1), updated.PlannedSignOffDate)
	assert.Equal(t, "520", updated.Allowances[2].MonthlyAmount.String())
	require.NoError(t, f.svc.Delete(ctx, k.PublicID))
	_, err = f.svc.Get(ctx, k.PublicID)
	requireAppErr(t, err, "not_found")
}

func TestCurrentAssignment(t *testing.T) {
	f := newContractFixture(t)
	ctx := context.Background()
	none, err := f.svc.CurrentAssignment(ctx, "CREWAAAA2")
	require.NoError(t, err)
	assert.Nil(t, none)

	in := f.input()
	in.SignOnDate = date(2026, 10, 1)
	k, err := f.svc.Create(ctx, in)
	require.NoError(t, err)
	_, err = f.svc.Activate(ctx, k.PublicID)
	require.NoError(t, err)

	a, err := f.svc.CurrentAssignment(ctx, "CREWAAAA2")
	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, 20, a.DaysOnBoard, "1 to 20 Oct in WIB inclusive")
	assert.Equal(t, date(2027, 4, 1), a.PlannedReliefDate, "falls back to planned sign-off")

	f.reliefs[k.ID] = date(2027, 3, 20)
	a, err = f.svc.CurrentAssignment(ctx, "CREWAAAA2")
	require.NoError(t, err)
	assert.Equal(t, date(2027, 3, 20), a.PlannedReliefDate)

	_, err = f.svc.CurrentAssignment(ctx, "MISSING22")
	requireAppErr(t, err, "not_found")
}
