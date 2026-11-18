package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

type payrollWorld struct {
	crewWorld
	crews        *repository.CrewRepository
	records      *repository.CrewRecordRepository
	payroll      *repository.PayrollRepository
	crewA, crewB uuid.UUID
	bni, bri     model.BankAccount
	mandiri      model.BankAccount
}

func setupPayrollWorld(t *testing.T) payrollWorld {
	t.Helper()
	w := payrollWorld{crewWorld: setupCrewWorld(t)}
	truncate(t, "payroll_periods")
	exec(t, `INSERT INTO currencies (code, decimals) VALUES ('IDR', 0)`)
	w.crews = repository.NewCrewRepository(testDB)
	w.records = repository.NewCrewRecordRepository(testDB)
	w.payroll = repository.NewPayrollRepository(testDB)
	w.crewA = createCrew(t, w.crews, "PAYAAAAA2", "Budi Santoso", nil, nil, "")
	w.crewB = createCrew(t, w.crews, "PAYBBBBB2", "Candra Wijaya", nil, nil, "")
	w.bni = w.account(t, w.crewA, "BANK NEGARA INDONESIA", "BUDI SANTOSO", "3200000001", true)
	w.bri = w.account(t, w.crewA, "BANK RAKYAT INDONESIA", "RINA SANTOSO", "100000000000001", false)
	w.mandiri = w.account(t, w.crewB, "MANDIRI", "SITI RAHAYU", "0123456789", true)
	return w
}

func (w payrollWorld) account(t *testing.T, crew uuid.UUID, bank, holder, number string, primary bool) model.BankAccount {
	t.Helper()
	a, err := w.records.SaveBankAccount(context.Background(), model.BankAccount{
		CrewID: crew, BankName: bank, AccountName: holder, AccountNumber: number, CurrencyCode: "IDR", IsPrimary: primary,
	})
	require.NoError(t, err)
	return a
}

func (w payrollWorld) period(t *testing.T, publicID string) model.PayrollPeriod {
	t.Helper()
	transfer := d(2026, 10, 20)
	p, err := w.payroll.CreatePeriod(context.Background(), model.PayrollPeriod{
		Title: "Salary September 2026", PeriodStart: d(2026, 8, 25), PeriodEnd: d(2026, 9, 24), TransferDate: &transfer,
	}, func() (string, error) { return publicID, nil }, 3)
	require.NoError(t, err)
	return p
}

type lineSpec struct {
	crew         uuid.UUID
	acc          model.BankAccount
	name, empNo  string
	rank, vessel string
	usd, idr     string
	remark       string
}

func (w payrollWorld) line(t *testing.T, periodID uuid.UUID, s lineSpec) model.PayrollLine {
	t.Helper()
	l := model.PayrollLine{
		PeriodID: periodID, CrewID: s.crew, BankAccountID: &s.acc.ID, EmployeeNo: s.empNo, CrewName: s.name,
		RankCode: s.rank, VesselName: s.vessel, BankName: s.acc.BankName, AccountName: s.acc.AccountName,
		AccountNumber: s.acc.AccountNumber, AmountIDR: decimal.RequireFromString(s.idr), TransferDate: d(2026, 10, 20), Remark: s.remark,
	}
	if s.usd != "" {
		usd := decimal.RequireFromString(s.usd)
		l.AmountUSD = &usd
	}
	saved, err := w.payroll.SaveLine(context.Background(), l)
	require.NoError(t, err)
	return saved
}

func (w payrollWorld) september(t *testing.T) model.PayrollPeriod {
	t.Helper()
	p := w.period(t, "PERAAAAA2")
	w.line(t, p.ID, lineSpec{crew: w.crewA, acc: w.bni, name: "BUDI SANTOSO", empNo: "900101", rank: "2E", vessel: "MV. NUSA 52", usd: "1800.00", idr: "32100000"})
	w.line(t, p.ID, lineSpec{crew: w.crewA, acc: w.bri, name: "BUDI SANTOSO", empNo: "900101", rank: "2E", vessel: "MV. NUSA 52", idr: "10000000", remark: "POTONG 10 JUTA - KE RINA SANTOSO"})
	w.line(t, p.ID, lineSpec{crew: w.crewB, acc: w.mandiri, name: "CANDRA WIJAYA", empNo: "900102", rank: "CE", vessel: "MV. NUSA FREEDOM", usd: "3900.00", idr: "70200000"})
	return p
}

func lineNames(page repository.Page[model.PayrollLine]) []string {
	out := make([]string, len(page.Items))
	for i, l := range page.Items {
		out[i] = l.AccountNumber
	}
	return out
}

func TestPayrollPeriod_CRUDAndTotals(t *testing.T) {
	w := setupPayrollWorld(t)
	ctx := context.Background()
	p := w.september(t)

	got, err := w.payroll.GetPeriod(ctx, "PERAAAAA2")
	require.NoError(t, err)
	assert.Equal(t, 3, got.LineCount)
	assert.Equal(t, "5700.00", got.TotalUSD.StringFixed(2))
	assert.Equal(t, "112300000.00", got.TotalIDR.StringFixed(2))
	require.NotNil(t, got.TransferDate)
	assert.Equal(t, d(2026, 10, 20), got.TransferDate.UTC())

	list, err := w.payroll.ListPeriods(ctx, "september", 1, 20)
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	assert.Equal(t, 3, list.Items[0].LineCount)

	page, err := w.payroll.ListLines(ctx, p.ID, model.PayrollLineFilter{}, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, []int{page.Items[0].LineNo, page.Items[1].LineNo, page.Items[2].LineNo})
	assert.Equal(t, "PAYAAAAA2", page.Items[0].CrewPublicID)
	assert.Nil(t, page.Items[1].AmountUSD)

	got.Title = "Salary Sep 2026"
	updated, err := w.payroll.UpdatePeriod(ctx, got)
	require.NoError(t, err)
	assert.Equal(t, "Salary Sep 2026", updated.Title)
	assert.Equal(t, 3, updated.LineCount)

	require.NoError(t, w.payroll.DeletePeriod(ctx, p.ID))
	_, err = w.payroll.GetPeriod(ctx, "PERAAAAA2")
	assert.ErrorIs(t, err, repository.ErrNotFound)
	var lines int
	require.NoError(t, testDB.QueryRow(ctx, `SELECT count(*) FROM payroll_lines`).Scan(&lines))
	assert.Zero(t, lines)
	assert.ErrorIs(t, w.payroll.DeletePeriod(ctx, p.ID), repository.ErrNotFound)
}

func TestPayrollLines_FiltersAndTotals(t *testing.T) {
	w := setupPayrollWorld(t)
	ctx := context.Background()
	p := w.september(t)
	list := func(f model.PayrollLineFilter) []string {
		page, err := w.payroll.ListLines(ctx, p.ID, f, 1, 20)
		require.NoError(t, err)
		assert.Len(t, page.Items, page.Total)
		return lineNames(page)
	}

	assert.Equal(t, []string{"0123456789"}, list(model.PayrollLineFilter{Banks: []string{"MANDIRI"}}))
	totals, err := w.payroll.LineTotals(ctx, p.ID, model.PayrollLineFilter{Banks: []string{"MANDIRI"}})
	require.NoError(t, err)
	assert.Equal(t, 1, totals.Count)
	assert.Equal(t, "3900.00", totals.USD.StringFixed(2))
	assert.Equal(t, "70200000.00", totals.IDR.StringFixed(2))

	assert.Equal(t, []string{"3200000001", "100000000000001"}, list(model.PayrollLineFilter{Q: "santoso"}))
	assert.Equal(t, []string{"0123456789"}, list(model.PayrollLineFilter{Q: "900102"}))
	assert.Equal(t, []string{"0123456789"}, list(model.PayrollLineFilter{Q: "01234"}))
	assert.Equal(t, []string{"100000000000001"}, list(model.PayrollLineFilter{Q: "potong"}))
	assert.Equal(t, []string{"100000000000001"}, list(model.PayrollLineFilter{HasRemark: true}))
	assert.Equal(t, []string{"100000000000001"}, list(model.PayrollLineFilter{NoUSD: true}))
	assert.Equal(t, []string{"0123456789"}, list(model.PayrollLineFilter{Vessels: []string{"MV. NUSA FREEDOM"}, Ranks: []string{"CE"}}))
	assert.Equal(t, []string{"0123456789", "3200000001", "100000000000001"}, list(model.PayrollLineFilter{Sort: "amount_idr", Desc: true}))
	assert.Equal(t, []string{"3200000001", "100000000000001", "0123456789"}, list(model.PayrollLineFilter{Sort: "unknown"}))

	all, err := w.payroll.AllLines(ctx, p.ID, model.PayrollLineFilter{Banks: []string{"BANK NEGARA INDONESIA", "MANDIRI"}})
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, "3200000001", all[0].AccountNumber)

	page, err := w.payroll.ListLines(ctx, p.ID, model.PayrollLineFilter{}, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, page.Total)
	assert.Equal(t, []string{"0123456789"}, lineNames(page))
}

func TestPayrollLines_FilterValuesWithCommas(t *testing.T) {
	w := setupPayrollWorld(t)
	ctx := context.Background()
	p := w.period(t, "PERCOMMA2")
	odd := w.account(t, w.crewB, "BANK A, TBK & CO", "CANDRA WIJAYA", "777", false)
	w.line(t, p.ID, lineSpec{crew: w.crewB, acc: odd, name: "CANDRA WIJAYA", idr: "1000"})

	page, err := w.payroll.ListLines(ctx, p.ID, model.PayrollLineFilter{Banks: []string{"BANK A, TBK & CO"}}, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 1, page.Total)
	page, err = w.payroll.ListLines(ctx, p.ID, model.PayrollLineFilter{Banks: []string{"BANK A"}}, 1, 20)
	require.NoError(t, err)
	assert.Zero(t, page.Total)
}

func TestPayrollFacets_DistinctSorted(t *testing.T) {
	w := setupPayrollWorld(t)
	p := w.september(t)
	w.line(t, p.ID, lineSpec{crew: w.crewB, acc: w.mandiri, name: "CANDRA WIJAYA", idr: "5000"})

	f, err := w.payroll.Facets(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"BANK NEGARA INDONESIA", "BANK RAKYAT INDONESIA", "MANDIRI"}, f.Banks)
	assert.Equal(t, []string{"MV. NUSA 52", "MV. NUSA FREEDOM"}, f.Vessels)
	assert.Equal(t, []string{"2E", "CE"}, f.Ranks)

	empty, err := w.payroll.Facets(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, empty.Banks)
}

func TestPayrollLine_KeepsSnapshotWhenAccountDeleted(t *testing.T) {
	w := setupPayrollWorld(t)
	ctx := context.Background()
	p := w.period(t, "PERSNAP02")
	l := w.line(t, p.ID, lineSpec{crew: w.crewA, acc: w.bri, name: "BUDI SANTOSO", idr: "10000000"})

	require.NoError(t, w.records.DeleteBankAccount(ctx, w.crewA, w.bri.ID))
	got, err := w.payroll.GetLine(ctx, l.ID)
	require.NoError(t, err)
	assert.Nil(t, got.BankAccountID)
	assert.Equal(t, "100000000000001", got.AccountNumber)

	got.Remark = "edited"
	got.AmountIDR = decimal.RequireFromString("9000000")
	saved, err := w.payroll.SaveLine(ctx, got)
	require.NoError(t, err)
	assert.Equal(t, "edited", saved.Remark)
	assert.Equal(t, 1, saved.LineNo)

	require.NoError(t, w.payroll.DeleteLine(ctx, l.ID))
	_, err = w.payroll.GetLine(ctx, l.ID)
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestPayrollImportLookups(t *testing.T) {
	w := setupPayrollWorld(t)
	ctx := context.Background()
	exec(t, `UPDATE crews SET employee_no = '0900101' WHERE id = $1`, w.crewA)
	exec(t, `INSERT INTO contracts (public_id, crew_id, vessel_id, rank_id, principal_id, employer_company_id, sign_on_date,
		duration_months, planned_sign_off_date, currency_code, basic_salary, status)
		VALUES ('KOLDAAAA2', $1, $2, $3, $4, $4, '2026-01-01', 6, '2026-06-30', 'USD', 1000, 'completed')`, w.crewA, w.vessel, w.ab, w.principalA)
	exec(t, `INSERT INTO contracts (public_id, crew_id, vessel_id, rank_id, principal_id, employer_company_id, sign_on_date,
		duration_months, planned_sign_off_date, actual_sign_off_date, currency_code, basic_salary, status, termination_reason)
		VALUES ('KTERAAAA2', $1, $2, $3, $4, $4, '2026-07-01', 6, '2026-12-31', '2026-09-05', 'USD', 1000, 'terminated', 'Medical')`,
		w.crewA, w.vessel, w.master, w.principalA)
	exec(t, `INSERT INTO contracts (public_id, crew_id, vessel_id, rank_id, principal_id, employer_company_id, sign_on_date,
		duration_months, planned_sign_off_date, currency_code, basic_salary, status)
		VALUES ('KDRFAAAA2', $1, $2, $3, $4, $4, '2026-09-01', 6, '2027-02-28', 'USD', 1000, 'draft')`, w.crewA, w.vessel, w.ab, w.principalA)

	crews, err := w.payroll.CrewsByEmployeeNo(ctx, []string{"900101", "999"})
	require.NoError(t, err)
	require.Len(t, crews, 1)
	assert.Equal(t, model.PayrollCrew{ID: w.crewA, PublicID: "PAYAAAAA2", FullName: "Budi Santoso", EmployeeNo: "0900101"}, crews["900101"])

	accounts, err := w.payroll.BankAccountsByCrew(ctx, []uuid.UUID{w.crewA, w.crewB})
	require.NoError(t, err)
	assert.Len(t, accounts[w.crewA], 2)
	assert.True(t, accounts[w.crewA][0].IsPrimary)
	assert.Len(t, accounts[w.crewB], 1)

	placements, err := w.payroll.CoveringContracts(ctx, []uuid.UUID{w.crewA, w.crewB}, d(2026, 8, 25), d(2026, 9, 24))
	require.NoError(t, err)
	assert.Equal(t, map[uuid.UUID]model.PayrollPlacement{w.crewA: {RankCode: "MST", VesselName: "MV Ocean"}}, placements)
}

func TestPayrollReplaceLines_AtomicReplaceAndNewAccounts(t *testing.T) {
	w := setupPayrollWorld(t)
	ctx := context.Background()
	p := w.september(t)
	newAccount := repository.PayrollNewAccount{Key: "k1", Account: model.BankAccount{
		CrewID: w.crewB, BankName: "BANK CENTRAL ASIA", AccountName: "RINA", AccountNumber: "5500000001", CurrencyCode: "IDR",
	}}
	staged := []repository.PayrollStagedLine{
		{Line: model.PayrollLine{CrewID: w.crewA, BankAccountID: &w.bni.ID, CrewName: "BUDI SANTOSO", BankName: w.bni.BankName,
			AccountName: w.bni.AccountName, AccountNumber: w.bni.AccountNumber, AmountIDR: decimal.RequireFromString("1"), TransferDate: d(2026, 10, 20)}},
		{NewAccountKey: "k1", Line: model.PayrollLine{CrewID: w.crewB, CrewName: "CANDRA WIJAYA", BankName: "BANK CENTRAL ASIA",
			AccountName: "RINA", AccountNumber: "5500000001", AmountIDR: decimal.RequireFromString("2"), TransferDate: d(2026, 10, 20)}},
	}

	added, err := w.payroll.ReplaceLines(ctx, p.ID, []repository.PayrollNewAccount{newAccount}, staged)
	require.NoError(t, err)
	assert.Equal(t, 1, added)
	page, err := w.payroll.ListLines(ctx, p.ID, model.PayrollLineFilter{}, 1, 20)
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
	assert.Equal(t, []int{1, 2}, []int{page.Items[0].LineNo, page.Items[1].LineNo})
	assert.Equal(t, p.ID, page.Items[1].PeriodID)
	accounts, err := w.records.ListBankAccounts(ctx, w.crewB)
	require.NoError(t, err)
	require.Len(t, accounts, 2)
	var created model.BankAccount
	for _, a := range accounts {
		if a.AccountNumber == "5500000001" {
			created = a
		}
	}
	require.NotNil(t, page.Items[1].BankAccountID)
	assert.Equal(t, created.ID, *page.Items[1].BankAccountID)

	require.NoError(t, w.payroll.DeletePeriod(ctx, p.ID))
	other := newAccount
	other.Account.AccountNumber = "1111111111"
	_, err = w.payroll.ReplaceLines(ctx, p.ID, []repository.PayrollNewAccount{other}, staged)
	assert.ErrorIs(t, err, repository.ErrNotFound)
	accounts, err = w.records.ListBankAccounts(ctx, w.crewB)
	require.NoError(t, err)
	assert.Len(t, accounts, 2)
}

func TestPayrollRepository_PeriodByIDAndCrewByPublicID(t *testing.T) {
	w := setupPayrollWorld(t)
	ctx := context.Background()
	p := w.september(t)

	byID, err := w.payroll.PeriodByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "PERAAAAA2", byID.PublicID)
	assert.Equal(t, 3, byID.LineCount)
	_, err = w.payroll.PeriodByID(ctx, uuid.New())
	assert.ErrorIs(t, err, repository.ErrNotFound)

	exec(t, `UPDATE crews SET employee_no = '900101' WHERE id = $1`, w.crewA)
	c, err := w.payroll.CrewByPublicID(ctx, "PAYAAAAA2")
	require.NoError(t, err)
	assert.Equal(t, model.PayrollCrew{ID: w.crewA, PublicID: "PAYAAAAA2", FullName: "Budi Santoso", EmployeeNo: "900101"}, c)
	_, err = w.payroll.CrewByPublicID(ctx, "NOBODY999")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestPayrollReplaceLines_ReusesAccountsWhenRunTwice(t *testing.T) {
	w := setupPayrollWorld(t)
	ctx := context.Background()
	p := w.period(t, "PERTWICE2")
	accounts := []repository.PayrollNewAccount{{Key: "k1", Account: model.BankAccount{
		CrewID: w.crewB, BankName: "BANK CENTRAL ASIA", AccountName: "RINA", AccountNumber: "5500 000-001", CurrencyCode: "IDR", IsPrimary: true,
	}}}
	staged := []repository.PayrollStagedLine{{NewAccountKey: "k1", Line: model.PayrollLine{
		CrewID: w.crewB, CrewName: "CANDRA WIJAYA", BankName: "BANK CENTRAL ASIA", AccountName: "RINA", AccountNumber: "5500 000-001",
		AmountIDR: decimal.RequireFromString("1"), TransferDate: d(2026, 10, 20),
	}}}

	added, err := w.payroll.ReplaceLines(ctx, p.ID, accounts, staged)
	require.NoError(t, err)
	assert.Equal(t, 1, added)
	added, err = w.payroll.ReplaceLines(ctx, p.ID, accounts, staged)
	require.NoError(t, err)
	assert.Zero(t, added)

	list, err := w.records.ListBankAccounts(ctx, w.crewB)
	require.NoError(t, err)
	require.Len(t, list, 2)
	primaries := 0
	for _, a := range list {
		if a.IsPrimary {
			primaries++
		}
	}
	assert.Equal(t, 1, primaries, "a staged primary flag never adds a second primary")
}
