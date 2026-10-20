package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

type contractWorld struct {
	crewWorld
	overtime, daily uuid.UUID
	crews           *repository.CrewRepository
	records         *repository.CrewRecordRepository
	contracts       *repository.ContractRepository
	plans           *repository.ReliefPlanRepository
}

func setupContractWorld(t *testing.T) contractWorld {
	t.Helper()
	w := contractWorld{crewWorld: setupCrewWorld(t)}
	truncate(t, "allowance_types")
	exec(t, `INSERT INTO currencies (code, decimals) VALUES ('IDR', 0)`)
	w.overtime = mustID(t, `INSERT INTO allowance_types (name, basis) VALUES ('Overtime', 'fixed_month') RETURNING id`)
	w.daily = mustID(t, `INSERT INTO allowance_types (name, basis) VALUES ('Offshore', 'fixed_day') RETURNING id`)
	exec(t, `UPDATE vessels SET grt = 1499.5, hp_kw = '5150 HP' WHERE id = $1`, w.vessel)
	w.crews = repository.NewCrewRepository(testDB)
	w.records = repository.NewCrewRecordRepository(testDB)
	w.contracts = repository.NewContractRepository(testDB)
	w.plans = repository.NewReliefPlanRepository(testDB)
	return w
}

func (w contractWorld) draft(crew uuid.UUID, currency string, signOn time.Time) model.Contract {
	return model.Contract{CrewID: crew, VesselID: w.vessel, RankID: w.master, PrincipalID: w.principalA, EmployerCompanyID: w.principalB,
		SignOnDate: signOn, DurationMonths: 6, PlannedSignOffDate: service.PlannedSignOff(signOn, 6), CurrencyCode: currency,
		BasicSalary: decimal.RequireFromString("5000"),
		Allowances: []model.ContractAllowance{
			{AllowanceTypeID: w.overtime, Basis: model.BasisFixedMonth, Value: decimal.RequireFromString("250.55"), MonthlyAmount: decimal.RequireFromString("250.55")},
			{AllowanceTypeID: w.daily, Basis: model.BasisFixedDay, Value: decimal.RequireFromString("10"), MonthlyAmount: decimal.RequireFromString("300")},
		}}
}

func (w contractWorld) create(t *testing.T, k model.Contract, pid string) model.Contract {
	t.Helper()
	id, err := w.contracts.Create(context.Background(), k, pid)
	require.NoError(t, err)
	got, err := w.contracts.GetByID(context.Background(), id)
	require.NoError(t, err)
	return got
}

func crewStatus(t *testing.T, id uuid.UUID) string {
	t.Helper()
	var code string
	require.NoError(t, testDB.QueryRow(context.Background(), `SELECT s.code FROM crews c JOIN crew_statuses s ON s.id = c.crew_status_id WHERE c.id = $1`, id).Scan(&code))
	return code
}

func TestContractRepositoryStoresAllowancesAndTotals(t *testing.T) {
	w := setupContractWorld(t)
	ctx := context.Background()
	crew := createCrew(t, w.crews, "CREWKKKK2", "Budi", &w.master, nil, "")

	k := w.create(t, w.draft(crew, "USD", d(2026, 11, 1)), "CONTRACT2")
	assert.Equal(t, model.ContractDraft, k.Status)
	assert.Equal(t, "Budi", k.CrewName)
	assert.Equal(t, "MV Ocean", k.VesselName)
	assert.Equal(t, 2, k.CurrencyDecimals)
	require.Len(t, k.Allowances, 2)
	assert.Equal(t, "Overtime", k.Allowances[0].AllowanceType)
	assert.True(t, decimal.RequireFromString("5550.55").Equal(k.TotalMonthlyPay))

	k.Allowances = k.Allowances[1:]
	k.Notes = "updated"
	require.NoError(t, w.contracts.Update(ctx, k, model.ContractDraft))
	assert.ErrorIs(t, w.contracts.Update(ctx, k, model.ContractActive), repository.ErrStateChanged)
	k, err := w.contracts.GetByPublicID(ctx, "CONTRACT2")
	require.NoError(t, err)
	assert.Len(t, k.Allowances, 1)
	assert.True(t, decimal.RequireFromString("5300").Equal(k.TotalMonthlyPay))

	idr := w.draft(crew, "IDR", d(2026, 11, 1))
	idr.BasicSalary = decimal.RequireFromString("12500000")
	idr.Allowances = nil
	got := w.create(t, idr, "CONTRACT3")
	assert.Equal(t, 0, got.CurrencyDecimals)
	assert.Equal(t, "12500000", got.BasicSalary.String())

	page, err := w.contracts.List(ctx, model.ContractFilter{Q: "budi", Sort: "basic_salary", Desc: true}, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
	assert.Equal(t, "CONTRACT3", page.Items[0].PublicID)

	before := d(2027, 6, 1)
	page, err = w.contracts.List(ctx, model.ContractFilter{Statuses: []string{"draft"}, CrewPublicID: "crewkkkk2", SignOffBefore: &before}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, page.Total)

	require.NoError(t, w.contracts.SoftDelete(ctx, got.ID))
	_, err = w.contracts.GetByPublicID(ctx, "CONTRACT3")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestActivationAndFinishUpdateCrewAndRecordOneExperience(t *testing.T) {
	w := setupContractWorld(t)
	ctx := context.Background()
	crew := createCrew(t, w.crews, "CREWKKKK2", "Budi", &w.master, nil, "")
	first := w.create(t, w.draft(crew, "USD", d(2026, 1, 1)), "CONTRACT2")
	second := w.create(t, w.draft(crew, "USD", d(2026, 2, 1)), "CONTRACT3")

	require.NoError(t, w.contracts.Activate(ctx, first.ID, crew))
	assert.Equal(t, "on_board", crewStatus(t, crew))
	assert.ErrorIs(t, w.contracts.Activate(ctx, second.ID, crew), repository.ErrCrewHasActiveContract, "second Active contract is rejected")
	assert.ErrorIs(t, w.contracts.Activate(ctx, first.ID, crew), repository.ErrStateChanged)

	active, err := w.contracts.ActiveForCrew(ctx, crew)
	require.NoError(t, err)
	assert.Equal(t, first.ID, active.ID)

	off := d(2026, 6, 30)
	x := service.ContractExperience(active, off, "")
	require.NoError(t, w.contracts.Finish(ctx, first.ID, crew, model.ContractCompleted, off, "", x))
	assert.Equal(t, "sign_off", crewStatus(t, crew))
	assert.ErrorIs(t, w.contracts.Finish(ctx, first.ID, crew, model.ContractCompleted, off, "", x), repository.ErrStateChanged)
	require.NoError(t, repository.RecordContractExperience(ctx, testDB, x), "re-recording is a no-op")

	exps, err := w.records.ListExperiences(ctx, crew)
	require.NoError(t, err)
	require.Len(t, exps, 1, "exactly one Experience per contract")
	assert.Equal(t, "contract", exps[0].Source)
	assert.Equal(t, "GRT 1,500 / 5150 HP", exps[0].GRTHP)
	assert.Equal(t, "Beta Marine", exps[0].CompanyName)
	assert.Equal(t, "Alpha Offshore", exps[0].PrincipalName)

	finished, err := w.contracts.GetByID(ctx, first.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ContractCompleted, finished.Status)
	assert.Equal(t, off, *finished.ActualSignOffDate)
}

func TestReliefPlansSuggestionsTimelineAndCompletion(t *testing.T) {
	w := setupContractWorld(t)
	ctx := context.Background()
	onBoard := createCrew(t, w.crews, "ONBOARD22", "On Board", &w.master, nil, "")
	veteran := createCrew(t, w.crews, "VETERAN22", "Veteran", &w.master, nil, "")
	rookie := createCrew(t, w.crews, "ROOKIE222", "Rookie", &w.master, nil, "")
	noCert := createCrew(t, w.crews, "NOCERT222", "No Cert", &w.master, nil, "")
	busy := createCrew(t, w.crews, "BUSYBUSY2", "Busy", &w.master, nil, "")
	rating := createCrew(t, w.crews, "RATING222", "Rating", &w.ab, nil, "")
	_ = rating

	valid := d(2030, 1, 1)
	for _, c := range []uuid.UUID{veteran, rookie, noCert, busy} {
		_, err := w.records.SaveDocument(ctx, repository.TravelDocuments, model.CrewDocument{CrewID: c, TypeID: w.passport, Number: "P-" + c.String()[:8], ExpiredDate: &valid})
		require.NoError(t, err)
	}
	for _, c := range []uuid.UUID{veteran, rookie, busy} {
		_, err := w.records.SaveDocument(ctx, repository.Certificates, model.CrewDocument{CrewID: c, TypeID: w.bst, Number: "B-" + c.String()[:8], ExpiredDate: &valid})
		require.NoError(t, err)
	}
	expOff := d(2025, 12, 31)
	_, err := w.records.SaveExperience(ctx, model.Experience{CrewID: veteran, VesselName: "MV", RankID: w.master, SignOnDate: d(2024, 1, 1), SignOffDate: &expOff, Status: "sign_off"})
	require.NoError(t, err)

	k := w.create(t, w.draft(onBoard, "USD", d(2026, 5, 1)), "CONTRACT2")
	require.NoError(t, w.contracts.Activate(ctx, k.ID, onBoard))
	other := w.create(t, w.draft(createCrew(t, w.crews, "OTHERCRW2", "Other", &w.master, nil, ""), "USD", d(2026, 6, 1)), "CONTRACT3")
	require.NoError(t, w.contracts.Activate(ctx, other.ID, other.CrewID))

	busyPlan, err := w.plans.Create(ctx, model.ReliefPlan{ContractID: other.ID, RelieverCrewID: busy, PlannedReliefDate: d(2026, 12, 1), Status: model.ReliefPlanned})
	require.NoError(t, err)

	relief := d(2026, 11, 1)
	list, err := w.plans.Suggestions(ctx, w.master, onBoard, relief, service.PlannedSignOff(relief, 6), 50)
	require.NoError(t, err)
	ids := make([]string, len(list))
	for i, c := range list {
		ids[i] = c.PublicID
	}
	assert.Equal(t, []string{"VETERAN22", "ROOKIE222"}, ids, "available, same rank, compliant, no overlapping relief, most sea service first")
	assert.Equal(t, 731, list[0].SeaServiceDays)

	windows, err := w.plans.OpenWindows(ctx, busy, uuid.Nil)
	require.NoError(t, err)
	require.Len(t, windows, 1)
	assert.Equal(t, 6, windows[0].DurationMonths)
	none, err := w.plans.OpenWindows(ctx, busy, busyPlan)
	require.NoError(t, err)
	assert.Empty(t, none)

	planID, err := w.plans.Create(ctx, model.ReliefPlan{ContractID: k.ID, RelieverCrewID: veteran, PlannedReliefDate: relief, JoiningPort: "Batam", Status: model.ReliefConfirmed})
	require.NoError(t, err)
	next, err := w.plans.NextReliefDate(ctx, k.ID)
	require.NoError(t, err)
	assert.Equal(t, relief, *next)

	bars, err := w.plans.TimelineBars(ctx, d(2026, 10, 1), d(2027, 1, 1), nil)
	require.NoError(t, err)
	assert.Len(t, bars, 2)
	open, err := w.plans.OpenPlansForContracts(ctx, []uuid.UUID{k.ID, other.ID})
	require.NoError(t, err)
	assert.Len(t, open, 2)
	lanes, err := w.plans.TimelineVessels(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, lanes, 1)

	ids2 := []string{"CONTRACT3", "NEWCONTR2"}
	newID := func() (string, error) {
		id := ids2[0]
		ids2 = ids2[1:]
		return id, nil
	}
	created, err := w.plans.Complete(ctx, planID, newID, 5)
	require.NoError(t, err)
	again, err := w.plans.Complete(ctx, planID, newID, 5)
	require.NoError(t, err)
	assert.Equal(t, created, again, "completing twice returns the same contract")

	nk, err := w.contracts.GetByID(ctx, created)
	require.NoError(t, err)
	assert.Equal(t, "NEWCONTR2", nk.PublicID, "retries after a public ID collision")
	assert.Equal(t, model.ContractDraft, nk.Status)
	assert.Equal(t, veteran, nk.CrewID)
	assert.Equal(t, relief, nk.SignOnDate)
	assert.Equal(t, d(2027, 5, 1), nk.PlannedSignOffDate)
	assert.Equal(t, "Batam", nk.SignOnPort)
	assert.Len(t, nk.Allowances, 2)
	assert.True(t, k.TotalMonthlyPay.Equal(nk.TotalMonthlyPay))

	plan, err := w.plans.Get(ctx, planID)
	require.NoError(t, err)
	assert.Equal(t, model.ReliefCompleted, plan.Status)
	assert.Equal(t, "NEWCONTR2", plan.NewContractPublicID)
	assert.ErrorIs(t, w.plans.Delete(ctx, planID), repository.ErrStateChanged)

	page, err := w.plans.List(ctx, model.ReliefPlanFilter{RelieverPublicID: "busybusy2"}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, page.Total)
}
