package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

type fakePayrollStore struct {
	period     model.PayrollPeriod
	periodGone bool
	crews      map[string]model.PayrollCrew
	accounts   map[uuid.UUID][]model.BankAccount
	placements map[uuid.UUID]model.PayrollPlacement

	replaceCalls int
	gotAccounts  []repository.PayrollNewAccount
	gotLines     []repository.PayrollStagedLine
}

func (f *fakePayrollStore) GetPeriod(_ context.Context, publicID string) (model.PayrollPeriod, error) {
	if f.periodGone || publicID != f.period.PublicID {
		return model.PayrollPeriod{}, fmt.Errorf("get payroll period: %w", repository.ErrNotFound)
	}
	return f.period, nil
}

func (f *fakePayrollStore) CrewsByEmployeeNo(_ context.Context, normalized []string) (map[string]model.PayrollCrew, error) {
	out := map[string]model.PayrollCrew{}
	for _, n := range normalized {
		if c, ok := f.crews[n]; ok {
			out[n] = c
		}
	}
	return out, nil
}

func (f *fakePayrollStore) BankAccountsByCrew(_ context.Context, crewIDs []uuid.UUID) (map[uuid.UUID][]model.BankAccount, error) {
	out := map[uuid.UUID][]model.BankAccount{}
	for _, id := range crewIDs {
		out[id] = f.accounts[id]
	}
	return out, nil
}

func (f *fakePayrollStore) CoveringContracts(_ context.Context, crewIDs []uuid.UUID, _, _ time.Time) (map[uuid.UUID]model.PayrollPlacement, error) {
	out := map[uuid.UUID]model.PayrollPlacement{}
	for _, id := range crewIDs {
		if p, ok := f.placements[id]; ok {
			out[id] = p
		}
	}
	return out, nil
}

func (f *fakePayrollStore) ReplaceLines(_ context.Context, _ uuid.UUID, accounts []repository.PayrollNewAccount, lines []repository.PayrollStagedLine) (int, error) {
	if f.periodGone {
		return 0, fmt.Errorf("replace payroll lines: %w", repository.ErrNotFound)
	}
	f.replaceCalls++
	f.gotAccounts, f.gotLines = accounts, lines
	return len(accounts), nil
}

type payrollImportFixture struct {
	svc          *PayrollImportService
	store        *fakePayrollStore
	cache        *memCache
	queue        *recordingQueue
	crewA, crewB model.PayrollCrew
}

func newPayrollImportFixture() payrollImportFixture {
	transfer := transferDay()
	crewA := model.PayrollCrew{ID: uuid.New(), PublicID: "CREWAAAA2", FullName: "Budi Santoso", EmployeeNo: "900101"}
	crewB := model.PayrollCrew{ID: uuid.New(), PublicID: "CREWBBBB2", FullName: "Candra Wijaya", EmployeeNo: "0900102"}
	store := &fakePayrollStore{
		period: model.PayrollPeriod{
			ID: uuid.New(), PublicID: "PERAAAAA2", Title: "Salary September 2026",
			PeriodStart: time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC), PeriodEnd: time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC),
			TransferDate: &transfer,
		},
		crews: map[string]model.PayrollCrew{"900101": crewA, "900102": crewB},
		accounts: map[uuid.UUID][]model.BankAccount{
			crewA.ID: {
				{ID: uuid.New(), CrewID: crewA.ID, BankName: "BANK NEGARA INDONESIA", AccountName: "BUDI SANTOSO", AccountNumber: "3200000001", IsPrimary: true},
				{ID: uuid.New(), CrewID: crewA.ID, BankName: "MANDIRI", AccountName: "SITI RAHAYU", AccountNumber: "0123456789"},
			},
		},
		placements: map[uuid.UUID]model.PayrollPlacement{crewA.ID: {RankCode: "2E", VesselName: "MV. NUSA 52"}},
	}
	cache, q := newMemCache(), &recordingQueue{}
	return payrollImportFixture{svc: NewPayrollImportService(store, cache, q), store: store, cache: cache, queue: q, crewA: crewA, crewB: crewB}
}

func (fx payrollImportFixture) validate(t *testing.T, rows ...wbRow) PayrollPreview {
	t.Helper()
	preview, err := fx.svc.Validate(context.Background(), "PERAAAAA2", buildPayrollWorkbook(t, wbOptions{}, rows...))
	require.NoError(t, err)
	return preview
}

func (fx payrollImportFixture) staged(t *testing.T, token string) stagedPayrollImport {
	t.Helper()
	raw, ok, err := fx.cache.Get(context.Background(), payrollImportKey(token))
	require.NoError(t, err)
	require.True(t, ok)
	var s stagedPayrollImport
	require.NoError(t, json.Unmarshal(raw, &s))
	return s
}

func byLine(p PayrollPreview) map[int]PayrollPreviewRow {
	out := map[int]PayrollPreviewRow{}
	for _, r := range p.Rows {
		out[r.Line] = r
	}
	return out
}

func TestPayrollImport_Statuses(t *testing.T) {
	fx := newPayrollImportFixture()
	fx.store.accounts[fx.crewB.ID] = []model.BankAccount{{ID: uuid.New(), CrewID: fx.crewB.ID, BankName: "BCA", AccountName: "ADE", AccountNumber: "555", IsPrimary: true}}
	p := fx.validate(t,
		wbRow{emp: 900101, account: "3200000001", usd: 1800, idr: 32100000},
		wbRow{emp: 900101, bank: "BANK CENTRAL ASIA", holder: "RINA", account: "5500000001", idr: 8000000},
		wbRow{emp: 999999, bank: "BNI", holder: "X", account: "1", idr: 1000},
		wbRow{emp: 900101, account: "3200000001"},
		wbRow{emp: 900102, account: "555", idr: 1000},
	)
	rows := byLine(p)
	assert.Equal(t, PayrollRowValid, rows[10].Status)
	assert.Empty(t, rows[10].Warnings)
	assert.Equal(t, "Budi Santoso", rows[10].CrewName)
	assert.Equal(t, PayrollRowNewAccount, rows[11].Status)
	assert.Equal(t, PayrollRowError, rows[12].Status)
	assert.Equal(t, "No crew member has this Employee ID.", rows[12].Errors["employee_no"])
	assert.Equal(t, PayrollRowError, rows[13].Status)
	assert.Equal(t, "IDR is required.", rows[13].Errors["idr"])
	assert.Equal(t, PayrollRowValid, rows[14].Status)
	assert.Equal(t, []string{"No contract covers this period; rank and vessel are left empty."}, rows[14].Warnings)

	assert.Equal(t, 5, p.Total)
	assert.Equal(t, 2, p.Valid)
	assert.Equal(t, 1, p.NewAccounts)
	assert.Equal(t, 2, p.Errors)
	assert.Equal(t, 1, p.WithWarnings)
}

func TestPayrollImport_AccountMatching(t *testing.T) {
	fx := newPayrollImportFixture()
	p := fx.validate(t,
		wbRow{emp: 900101, account: "3200 000-001", idr: 1},
		wbRow{emp: 900101, account: int64(123456789), idr: 2},
		wbRow{emp: 900101, bank: "BANK RAKYAT INDONESIA", holder: "FAJAR NUGROHO", account: int64(400000002), idr: 3},
		wbRow{emp: 900101, bank: "BANK CENTRAL ASIA", holder: "RINA", account: "5500000001", idr: 4},
		wbRow{emp: 900101, bank: "BANK CENTRAL ASIA", holder: "RINA", account: "5500000001", idr: 5},
	)
	require.Zero(t, p.Errors, "%+v", p.Rows)
	rows := byLine(p)
	assert.Equal(t, PayrollRowValid, rows[10].Status)
	assert.Equal(t, PayrollRowValid, rows[11].Status)
	assert.Equal(t, "0123456789", rows[11].AccountNumber)
	assert.Equal(t, PayrollRowNewAccount, rows[12].Status)
	assert.Equal(t, []string{"The account number was stored as a number in the file; check that no leading zero is missing."}, rows[12].Warnings)

	s := fx.staged(t, p.Token)
	require.Len(t, s.Lines, 5)
	assert.Equal(t, "3200000001", s.Lines[0].Line.AccountNumber)
	assert.Equal(t, fx.store.accounts[fx.crewA.ID][0].ID, *s.Lines[0].Line.BankAccountID)
	assert.Equal(t, "0123456789", s.Lines[1].Line.AccountNumber)
	assert.Equal(t, "MANDIRI", s.Lines[1].Line.BankName)
	require.Len(t, s.Accounts, 2)
	assert.Equal(t, s.Lines[3].NewAccountKey, s.Lines[4].NewAccountKey)
	assert.Nil(t, s.Lines[3].Line.BankAccountID)
	assert.Equal(t, "2E", s.Lines[0].Line.RankCode)
	assert.Equal(t, "MV. NUSA 52", s.Lines[0].Line.VesselName)
	assert.Equal(t, "900101", s.Lines[0].Line.EmployeeNo)
}

func TestPayrollImport_NewAccountPrimaryOnlyWhenNoneExists(t *testing.T) {
	fx := newPayrollImportFixture()
	p := fx.validate(t,
		wbRow{emp: 900102, bank: "BCA", holder: "ADE", account: "111", idr: 1},
		wbRow{emp: 900102, bank: "BRI", holder: "IBU", account: "222", idr: 2},
		wbRow{emp: 900101, bank: "BRI", holder: "RINA", account: "333", idr: 3},
	)
	s := fx.staged(t, p.Token)
	require.Len(t, s.Accounts, 3)
	primary := map[string]bool{}
	for _, a := range s.Accounts {
		primary[a.Account.AccountNumber] = a.Account.IsPrimary
		assert.Equal(t, "IDR", a.Account.CurrencyCode)
	}
	assert.Equal(t, map[string]bool{"111": true, "222": false, "333": false}, primary)
	assert.Equal(t, "0900102", s.Lines[0].Line.EmployeeNo)
}

func TestPayrollImport_NewAccountNeedsBankAndHolder(t *testing.T) {
	fx := newPayrollImportFixture()
	p := fx.validate(t, wbRow{emp: 900101, account: "999", idr: 1})
	row := byLine(p)[10]
	assert.Equal(t, PayrollRowError, row.Status)
	assert.Equal(t, "Bank name is required for a new account.", row.Errors["bank_name"])
	assert.Equal(t, "Account name is required for a new account.", row.Errors["account_name"])
}

func TestPayrollImport_BlankDateUsesPeriodTransferDate(t *testing.T) {
	fx := newPayrollImportFixture()
	p := fx.validate(t, wbRow{emp: 900101, account: "3200000001", idr: 1})
	s := fx.staged(t, p.Token)
	assert.Equal(t, transferDay(), s.Lines[0].Line.TransferDate.UTC())
	assert.Equal(t, "2026-10-20", byLine(p)[10].TransferDate)

	fx.store.period.TransferDate = nil
	p = fx.validate(t, wbRow{emp: 900101, account: "3200000001", idr: 1})
	assert.Equal(t, "A date is required when the period has no transfer date.", byLine(p)[10].Errors["date"])
}

func TestPayrollImport_NegativeAmounts(t *testing.T) {
	fx := newPayrollImportFixture()
	p := fx.validate(t, wbRow{emp: 900101, account: "3200000001", usd: -1, idr: -5})
	row := byLine(p)[10]
	assert.Equal(t, "USD cannot be negative.", row.Errors["usd"])
	assert.Equal(t, "IDR cannot be negative.", row.Errors["idr"])
}

func TestPayrollImport_NoTokenWhileErrors(t *testing.T) {
	fx := newPayrollImportFixture()
	p := fx.validate(t, wbRow{emp: 999999, account: "1", idr: 1})
	assert.Empty(t, p.Token)
	_, err := fx.svc.Commit(context.Background(), "PERAAAAA2", "")
	requireAppError(t, err, 410, "import_expired")
}

func TestPayrollImport_RejectsNonWorkbook(t *testing.T) {
	fx := newPayrollImportFixture()
	_, err := fx.svc.Validate(context.Background(), "PERAAAAA2", []byte("Employee ID,IDR\n1,2\n"))
	requireAppError(t, err, 422, "file_rejected")
	_, err = fx.svc.Validate(context.Background(), "UNKNOWN99", buildPayrollWorkbook(t, wbOptions{}, wbRow{emp: 1, account: "1", idr: 1}))
	requireAppError(t, err, 404, "not_found")
}

func TestPayrollImport_WillReplaceAndCounts(t *testing.T) {
	fx := newPayrollImportFixture()
	fx.store.period.LineCount = 91
	p := fx.validate(t, wbRow{emp: 900101, account: "3200000001", idr: 1}, wbRow{emp: 900101, account: "0123456789", idr: 2})
	assert.Equal(t, 91, p.WillReplace)
	assert.Equal(t, 2, p.Total)
}

func TestPayrollImport_CommitAndHandleReplaceThePeriod(t *testing.T) {
	fx := newPayrollImportFixture()
	p := fx.validate(t,
		wbRow{emp: 900101, account: "3200000001", usd: 1800, idr: 32100000},
		wbRow{emp: 900102, bank: "BCA", holder: "ADE", account: "111", idr: 2},
	)
	j, err := fx.svc.Commit(context.Background(), "PERAAAAA2", p.Token)
	require.NoError(t, err)
	assert.Equal(t, PayrollImportJob, fx.queue.kind)

	payload, err := json.Marshal(fx.queue.payload)
	require.NoError(t, err)
	j.Payload = payload
	res, err := fx.svc.Handle(context.Background(), j)
	require.NoError(t, err)
	assert.Equal(t, PayrollImportReport{Lines: 2, AccountsAdded: 1}, res)
	assert.Equal(t, 1, fx.store.replaceCalls)
	assert.Equal(t, "1800", fx.store.gotLines[0].Line.AmountUSD.String())
	_, ok, err := fx.cache.Get(context.Background(), payrollImportKey(p.Token))
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestPayrollImport_CommitAfterPeriodDeleted(t *testing.T) {
	fx := newPayrollImportFixture()
	p := fx.validate(t, wbRow{emp: 900101, account: "3200000001", idr: 1})
	fx.store.periodGone = true
	_, err := fx.svc.Commit(context.Background(), "PERAAAAA2", p.Token)
	requireAppError(t, err, 404, "not_found")

	payload, err := json.Marshal(map[string]string{"token": p.Token})
	require.NoError(t, err)
	_, err = fx.svc.Handle(context.Background(), model.Job{Payload: payload})
	require.Error(t, err)

	require.NoError(t, fx.cache.Delete(context.Background(), payrollImportKey(p.Token)))
	_, err = fx.svc.Handle(context.Background(), model.Job{Payload: payload})
	require.Error(t, err)
	assert.Zero(t, fx.store.replaceCalls)
}

func TestPayrollImport_TokenBoundToPeriod(t *testing.T) {
	fx := newPayrollImportFixture()
	p := fx.validate(t, wbRow{emp: 900101, account: "3200000001", idr: 1})
	_, err := fx.svc.Commit(context.Background(), "PEROTHER2", p.Token)
	requireAppError(t, err, 410, "import_expired")
}

func TestPayrollImport_KeylessAmountRowIsAnError(t *testing.T) {
	fx := newPayrollImportFixture()
	p := fx.validate(t, wbRow{emp: 900101, account: "3200000001", idr: 1}, wbRow{idr: 5000})
	row := byLine(p)[11]
	assert.Equal(t, PayrollRowError, row.Status)
	assert.Equal(t, "The Employee ID is required.", row.Errors["employee_no"])
	assert.Empty(t, p.Token)
}
