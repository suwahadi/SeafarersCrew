package repository_test

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

type crewWorld struct {
	master, ab                uuid.UUID
	passport, seamanBook, bst uuid.UUID
	principalA, principalB    uuid.UUID
	vessel                    uuid.UUID
	today                     time.Time
}

func d(y int, m time.Month, day int) time.Time { return time.Date(y, m, day, 0, 0, 0, 0, time.UTC) }

func mustID(t *testing.T, sql string, args ...any) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, testDB.QueryRow(context.Background(), sql, args...).Scan(&id))
	return id
}

func exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	_, err := testDB.Exec(context.Background(), sql, args...)
	require.NoError(t, err)
}

// setupCrewWorld seeds the minimum master data used by the crew tests.
func setupCrewWorld(t *testing.T) crewWorld {
	t.Helper()
	truncate(t, "crews", "contracts", "vessels", "companies", "ranks", "crew_statuses", "currencies", "document_types", "certificate_types", "vessel_types")
	for i, code := range []string{"available", "on_board", "sign_off", "medical_leave", "blacklisted"} {
		exec(t, `INSERT INTO crew_statuses (code, name, is_system, sort_order) VALUES ($1, $1, true, $2)`, code, i)
	}
	exec(t, `INSERT INTO currencies (code, decimals) VALUES ('USD', 2)`)
	w := crewWorld{today: d(2026, 10, 20)}
	w.master = mustID(t, `INSERT INTO ranks (name, short_code, department, sort_order) VALUES ('Master', 'MST', 'deck', 1) RETURNING id`)
	w.ab = mustID(t, `INSERT INTO ranks (name, short_code, department, sort_order) VALUES ('AB', 'AB', 'deck', 2) RETURNING id`)
	w.passport = mustID(t, `INSERT INTO document_types (code, name, has_expiry) VALUES ('passport', 'Passport', true) RETURNING id`)
	w.seamanBook = mustID(t, `INSERT INTO document_types (code, name, has_expiry) VALUES ('seaman_book', 'Seaman Book', true) RETURNING id`)
	w.bst = mustID(t, `INSERT INTO certificate_types (code, name, category, has_expiry) VALUES ('bst', 'BST', 'cop', true) RETURNING id`)
	exec(t, `INSERT INTO compliance_requirements (rank_id, requirement_type, document_type_id) VALUES ($1, 'document', $2)`, w.master, w.passport)
	exec(t, `INSERT INTO compliance_requirements (rank_id, requirement_type, certificate_type_id) VALUES ($1, 'certificate', $2)`, w.master, w.bst)
	w.principalA = mustID(t, `INSERT INTO companies (code, name, type) VALUES ('A', 'Alpha Offshore', 'principal') RETURNING id`)
	w.principalB = mustID(t, `INSERT INTO companies (code, name, type) VALUES ('B', 'Beta Marine', 'both') RETURNING id`)
	w.vessel = mustID(t, `INSERT INTO vessels (name, imo_number) VALUES ('MV Ocean', '9074729') RETURNING id`)
	return w
}

func createCrew(t *testing.T, repo *repository.CrewRepository, pid, name string, rank *uuid.UUID, dob *time.Time, nationality string) uuid.UUID {
	t.Helper()
	id, err := repo.Create(context.Background(), model.Crew{FullName: name, AppliedRankID: rank, DateOfBirth: dob, Nationality: nationality}, pid, "available")
	require.NoError(t, err)
	return id
}

func listIDs(t *testing.T, repo *repository.CrewRepository, f model.CrewFilter) []string {
	t.Helper()
	page, _, err := repo.ListCrews(context.Background(), f, 100, 0)
	require.NoError(t, err)
	ids := make([]string, len(page.Items))
	for i, it := range page.Items {
		ids[i] = it.PublicID
	}
	assert.Equal(t, len(ids), page.Total)
	return ids
}

func TestCrewRepositoryCRUDAndStatus(t *testing.T) {
	w := setupCrewWorld(t)
	ctx := context.Background()
	repo := repository.NewCrewRepository(testDB)
	id := createCrew(t, repo, "AAAAAAAA2", "Budi Santoso", &w.master, nil, "Indonesian")

	c, err := repo.GetByPublicID(ctx, "AAAAAAAA2")
	require.NoError(t, err)
	assert.Equal(t, "available", c.StatusCode)
	assert.Equal(t, "Master", c.AppliedRankName)

	c.FullName = "Budi S."
	require.NoError(t, repo.Update(ctx, id, c))
	require.NoError(t, repo.SetStatus(ctx, id, "blacklisted", "Fraud"))
	c, err = repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "Budi S.", c.FullName)
	assert.Equal(t, "blacklisted", c.StatusCode)
	assert.Equal(t, "Fraud", c.StatusReason)

	rank, err := repo.ActiveContractRank(ctx, id)
	require.NoError(t, err)
	assert.Nil(t, rank)

	require.NoError(t, repo.SoftDelete(ctx, id))
	_, err = repo.GetByPublicID(ctx, "AAAAAAAA2")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestCrewDatagridSearchAndFilters(t *testing.T) {
	w := setupCrewWorld(t)
	ctx := context.Background()
	repo := repository.NewCrewRepository(testDB)
	records := repository.NewCrewRecordRepository(testDB)
	dob := func(y int) *time.Time { v := d(y, 6, 1); return &v }

	compliant := createCrew(t, repo, "CCCCCCCC2", "Andi Pratama", &w.master, dob(1980), "Indonesian")
	expired := createCrew(t, repo, "EEEEEEEE2", "Joko Widodo", &w.master, dob(1995), "Indonesian")
	rating := createCrew(t, repo, "RRRRRRRR2", "Maria Santos", &w.ab, dob(2000), "Filipino")

	saveDoc := func(kind repository.DocumentKind, crew, typ uuid.UUID, number string, exp time.Time) {
		_, err := records.SaveDocument(ctx, kind, model.CrewDocument{CrewID: crew, TypeID: typ, Number: number, ExpiredDate: &exp})
		require.NoError(t, err)
	}
	saveDoc(repository.TravelDocuments, compliant, w.passport, "X1234567", d(2030, 1, 1))
	saveDoc(repository.TravelDocuments, compliant, w.passport, "OLD00001", d(2020, 1, 1))
	saveDoc(repository.Certificates, compliant, w.bst, "BST-1", d(2026, 11, 5))
	saveDoc(repository.TravelDocuments, expired, w.passport, "P7777777", d(2029, 1, 1))
	saveDoc(repository.Certificates, expired, w.bst, "BST-2", d(2026, 10, 1))
	saveDoc(repository.TravelDocuments, rating, w.seamanBook, "SB-555", d(2031, 1, 1))

	base := model.CrewFilter{Today: w.today}
	with := func(mut func(*model.CrewFilter)) model.CrewFilter {
		f := base
		mut(&f)
		return f
	}

	assert.ElementsMatch(t, []string{"CCCCCCCC2", "EEEEEEEE2", "RRRRRRRR2"}, listIDs(t, repo, base))
	assert.Equal(t, []string{"CCCCCCCC2"}, listIDs(t, repo, with(func(f *model.CrewFilter) { f.Q = "andi" })))
	assert.Equal(t, []string{"EEEEEEEE2"}, listIDs(t, repo, with(func(f *model.CrewFilter) { f.Q = "eeeeeeee2" })))
	assert.Equal(t, []string{"CCCCCCCC2"}, listIDs(t, repo, with(func(f *model.CrewFilter) { f.Q = "X12345" })), "passport number")
	assert.Equal(t, []string{"RRRRRRRR2"}, listIDs(t, repo, with(func(f *model.CrewFilter) { f.Q = "sb-555" })), "seaman book number")
	assert.Empty(t, listIDs(t, repo, with(func(f *model.CrewFilter) { f.Q = "%" })), "wildcards are escaped")

	assert.Equal(t, []string{"RRRRRRRR2"}, listIDs(t, repo, with(func(f *model.CrewFilter) { f.RankIDs = []uuid.UUID{w.ab} })))
	assert.Equal(t, []string{"RRRRRRRR2"}, listIDs(t, repo, with(func(f *model.CrewFilter) { f.Nationalities = []string{"FILIPINO"} })))
	minAge, maxAge := 30, 40
	assert.Equal(t, []string{"EEEEEEEE2"}, listIDs(t, repo, with(func(f *model.CrewFilter) { f.AgeMin, f.AgeMax = &minAge, &maxAge }))) // 31 years old

	assert.ElementsMatch(t, []string{"CCCCCCCC2", "RRRRRRRR2"}, listIDs(t, repo, with(func(f *model.CrewFilter) { f.Compliance = "compliant" })),
		"a rank without matrix entries has nothing mandatory, matching service.EvaluateCompliance")
	assert.Equal(t, []string{"EEEEEEEE2"}, listIDs(t, repo, with(func(f *model.CrewFilter) { f.Compliance = "not_compliant" })),
		"the expired BST fails the Master rank")

	assert.Equal(t, []string{"EEEEEEEE2"}, listIDs(t, repo, with(func(f *model.CrewFilter) { f.DocValidity = "expired" })),
		"the renewed passport hides the older expired one")
	assert.Equal(t, []string{"CCCCCCCC2"}, listIDs(t, repo, with(func(f *model.CrewFilter) { f.DocValidity = "expiring"; f.DocDays = 30 })))
	assert.Equal(t, []string{"RRRRRRRR2"}, listIDs(t, repo, with(func(f *model.CrewFilter) { f.DocValidity = "valid" })))

	require.NoError(t, repo.SetStatus(ctx, rating, "blacklisted", "x"))
	assert.Equal(t, []string{"RRRRRRRR2"}, listIDs(t, repo, with(func(f *model.CrewFilter) { f.StatusCodes = []string{"blacklisted"} })))
	assert.ElementsMatch(t, []string{"CCCCCCCC2", "EEEEEEEE2"}, listIDs(t, repo, with(func(f *model.CrewFilter) { f.ExcludeStatuses = []string{"blacklisted"} })))
}

func TestCrewDatagridSeaServiceGroupingAndSort(t *testing.T) {
	w := setupCrewWorld(t)
	ctx := context.Background()
	repo := repository.NewCrewRepository(testDB)
	records := repository.NewCrewRecordRepository(testDB)

	veteran := createCrew(t, repo, "VVVVVVVV2", "Veteran", &w.master, nil, "")
	rookie := createCrew(t, repo, "KKKKKKKK2", "Rookie", &w.master, nil, "")
	addExp := func(crew uuid.UUID, on, off time.Time) {
		_, err := records.SaveExperience(ctx, model.Experience{CrewID: crew, VesselName: "MV", RankID: w.master, SignOnDate: on, SignOffDate: &off, Status: "sign_off"})
		require.NoError(t, err)
	}
	addExp(veteran, d(2024, 1, 1), d(2024, 12, 31))
	addExp(veteran, d(2024, 6, 1), d(2025, 3, 31))
	addExp(rookie, d(2025, 1, 1), d(2025, 1, 31))

	f := model.CrewFilter{Today: w.today, SeaRankID: &w.master, SeaMinDays: 400}
	assert.Equal(t, []string{"VVVVVVVV2"}, listIDs(t, repo, f), "455 merged days, not 579")

	f = model.CrewFilter{Today: w.today, Sort: "sea_service", Desc: true}
	page, _, err := repo.ListCrews(ctx, f, 10, 0)
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	assert.Equal(t, "VVVVVVVV2", page.Items[0].PublicID)
	assert.Equal(t, 456, page.Items[0].SeaServiceDays)
	assert.Equal(t, 31, page.Items[1].SeaServiceDays)

	exec(t, `INSERT INTO contracts (public_id, crew_id, vessel_id, rank_id, principal_id, employer_company_id, sign_on_date,
		duration_months, planned_sign_off_date, currency_code, basic_salary, status)
		VALUES ('CONTRACT2', $1, $2, $3, $4, $5, '2026-09-01', 6, '2027-03-01', 'USD', 5000, 'active')`, rookie, w.vessel, w.master, w.principalA, w.principalB)

	page, groups, err := repo.ListCrews(ctx, model.CrewFilter{Today: w.today, GroupBy: "principal"}, 10, 0)
	require.NoError(t, err)
	require.Len(t, groups, 2)
	assert.Equal(t, "", groups[0].Name, "crew without an active contract")
	assert.Equal(t, "Alpha Offshore", groups[1].Name)
	assert.Equal(t, 1, groups[1].Count)
	assert.Equal(t, "VVVVVVVV2", page.Items[0].PublicID)
	assert.Equal(t, "MV Ocean", page.Items[1].VesselName)
	assert.Equal(t, "Beta Marine", page.Items[1].EmployerName)

	assert.Equal(t, []string{"KKKKKKKK2"}, listIDs(t, repo, model.CrewFilter{Today: w.today, PrincipalIDs: []uuid.UUID{w.principalA}}))
	assert.Equal(t, []string{"KKKKKKKK2"}, listIDs(t, repo, model.CrewFilter{Today: w.today, VesselIDs: []uuid.UUID{w.vessel}}))

	var streamed []string
	require.NoError(t, repo.StreamCrews(ctx, model.CrewFilter{Today: w.today, Sort: "full_name"}, 1, func(r model.CrewRow) error {
		streamed = append(streamed, r.FullName)
		return nil
	}))
	assert.Equal(t, []string{"Rookie", "Veteran"}, streamed)
}

func TestCrewRecordsKeepSinglePrimary(t *testing.T) {
	w := setupCrewWorld(t)
	ctx := context.Background()
	repo := repository.NewCrewRepository(testDB)
	records := repository.NewCrewRecordRepository(testDB)
	crew := createCrew(t, repo, "PPPPPPPP2", "Primary Test", &w.master, nil, "")

	a, err := records.SaveBankAccount(ctx, model.BankAccount{CrewID: crew, BankName: "BCA", AccountName: "P", AccountNumber: "1", CurrencyCode: "USD", IsPrimary: true})
	require.NoError(t, err)
	_, err = records.SaveBankAccount(ctx, model.BankAccount{CrewID: crew, BankName: "BNI", AccountName: "P", AccountNumber: "2", CurrencyCode: "USD", IsPrimary: true})
	require.NoError(t, err)
	list, err := records.ListBankAccounts(ctx, crew)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "BNI", list[0].BankName)
	assert.False(t, list[1].IsPrimary)

	err = records.DeleteBankAccount(ctx, uuid.New(), a.ID)
	assert.ErrorIs(t, err, repository.ErrNotFound, "rows are scoped to their crew")
}

func TestImportCrewsIsolatesFailedRows(t *testing.T) {
	w := setupCrewWorld(t)
	ctx := context.Background()
	repo := repository.NewCrewRepository(testDB)

	l, err := repo.ImportLookups(ctx)
	require.NoError(t, err)
	assert.Equal(t, w.master, l.Ranks["MST"])
	assert.Equal(t, w.master, l.Ranks["MASTER"])
	assert.Equal(t, w.passport, l.PassportType)

	exp := d(2030, 1, 1)
	ids := []string{"IMPORTAA2", "IMPORTAA2", "IMPORTBB2", "IMPORTCC2"}
	next := func() (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	}
	outcomes, err := repo.ImportCrews(ctx, []repository.ImportCrew{
		{Line: 2, Crew: model.Crew{FullName: "First"}, Documents: []repository.ImportDocument{{TypeID: l.PassportType, Number: "Z9999999", ExpiredDate: &exp}}},
		{Line: 3, Crew: model.Crew{FullName: "Collides then succeeds"}},
		{Line: 4, Crew: model.Crew{FullName: "Unknown document type"}, Documents: []repository.ImportDocument{{TypeID: uuid.New(), Number: "Q1", ExpiredDate: &exp}}},
	}, next, 5)
	require.NoError(t, err)
	require.Len(t, outcomes, 3)
	assert.Equal(t, "IMPORTAA2", outcomes[0].PublicID)
	assert.Equal(t, "IMPORTBB2", outcomes[1].PublicID, "retries after a public ID collision")
	assert.NotEmpty(t, outcomes[2].Error)

	existing, err := repo.ExistingDocumentNumbers(ctx, l.PassportType, []string{"Z9999999", "NOPE"})
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"Z9999999": true}, existing)

	var count int
	require.NoError(t, testDB.QueryRow(ctx, `SELECT count(*) FROM crews`).Scan(&count))
	assert.Equal(t, 2, count)
}
