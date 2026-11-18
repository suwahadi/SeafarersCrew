package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

type fakePayrollService struct {
	fakePayrollStore
	lines       map[uuid.UUID]model.PayrollLine
	saved       []model.PayrollLine
	pageFilter  model.PayrollLineFilter
	totalFilter model.PayrollLineFilter
	crewsByPub  map[string]model.PayrollCrew
}

func (f *fakePayrollService) CreatePeriod(_ context.Context, p model.PayrollPeriod, newID func() (string, error), _ int) (model.PayrollPeriod, error) {
	id, err := newID()
	if err != nil {
		return p, err
	}
	p.ID, p.PublicID = uuid.New(), id
	f.period = p
	return p, nil
}

func (f *fakePayrollService) PeriodByID(_ context.Context, id uuid.UUID) (model.PayrollPeriod, error) {
	if id != f.period.ID {
		return model.PayrollPeriod{}, repository.ErrNotFound
	}
	return f.period, nil
}

func (f *fakePayrollService) ListPeriods(context.Context, string, int, int) (repository.Page[model.PayrollPeriod], error) {
	return repository.Page[model.PayrollPeriod]{Items: []model.PayrollPeriod{f.period}, Total: 1}, nil
}

func (f *fakePayrollService) UpdatePeriod(_ context.Context, p model.PayrollPeriod) (model.PayrollPeriod, error) {
	f.period = p
	return p, nil
}

func (f *fakePayrollService) DeletePeriod(context.Context, uuid.UUID) error { return nil }

func (f *fakePayrollService) ListLines(_ context.Context, _ uuid.UUID, filter model.PayrollLineFilter, _, _ int) (repository.Page[model.PayrollLine], error) {
	f.pageFilter = filter
	return repository.Page[model.PayrollLine]{Total: 3}, nil
}

func (f *fakePayrollService) LineTotals(_ context.Context, _ uuid.UUID, filter model.PayrollLineFilter) (model.PayrollTotals, error) {
	f.totalFilter = filter
	return model.PayrollTotals{Count: 3, IDR: decimal.RequireFromString("112320000")}, nil
}

func (f *fakePayrollService) Facets(context.Context, uuid.UUID) (model.PayrollFacets, error) {
	return model.PayrollFacets{Banks: []string{"MANDIRI"}}, nil
}

func (f *fakePayrollService) GetLine(_ context.Context, id uuid.UUID) (model.PayrollLine, error) {
	l, ok := f.lines[id]
	if !ok {
		return l, fmt.Errorf("get payroll line: %w", repository.ErrNotFound)
	}
	return l, nil
}

func (f *fakePayrollService) SaveLine(_ context.Context, l model.PayrollLine) (model.PayrollLine, error) {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	f.saved = append(f.saved, l)
	return l, nil
}

func (f *fakePayrollService) DeleteLine(context.Context, uuid.UUID) error { return nil }

func (f *fakePayrollService) CrewByPublicID(_ context.Context, publicID string) (model.PayrollCrew, error) {
	c, ok := f.crewsByPub[publicID]
	if !ok {
		return c, fmt.Errorf("get crew: %w", repository.ErrNotFound)
	}
	return c, nil
}

func newPayrollServiceFixture() (*PayrollService, *fakePayrollService, payrollImportFixture) {
	fx := newPayrollImportFixture()
	store := &fakePayrollService{
		fakePayrollStore: *fx.store,
		lines:            map[uuid.UUID]model.PayrollLine{},
		crewsByPub:       map[string]model.PayrollCrew{fx.crewA.PublicID: fx.crewA, fx.crewB.PublicID: fx.crewB},
	}
	return NewPayrollService(store), store, fx
}

func TestPayrollService_CreateLineFillsSnapshots(t *testing.T) {
	svc, store, fx := newPayrollServiceFixture()
	account := store.accounts[fx.crewA.ID][1]
	l, err := svc.CreateLine(context.Background(), "PERAAAAA2", model.PayrollLineInput{
		CrewPublicID: fx.crewA.PublicID, BankAccountID: account.ID, AmountIDR: decimal.RequireFromString("8000000"), Remark: "8 juta",
	})
	require.NoError(t, err)
	assert.Equal(t, store.period.ID, l.PeriodID)
	assert.Equal(t, fx.crewA.ID, l.CrewID)
	assert.Equal(t, "Budi Santoso", l.CrewName)
	assert.Equal(t, "900101", l.EmployeeNo)
	assert.Equal(t, "2E", l.RankCode)
	assert.Equal(t, "MV. NUSA 52", l.VesselName)
	assert.Equal(t, "MANDIRI", l.BankName)
	assert.Equal(t, "0123456789", l.AccountNumber)
	assert.Equal(t, account.ID, *l.BankAccountID)
	assert.Equal(t, transferDay(), l.TransferDate.UTC())
}

func TestPayrollService_CreateLineRejectsForeignAccountAndUnknownCrew(t *testing.T) {
	svc, store, fx := newPayrollServiceFixture()
	foreign := store.accounts[fx.crewA.ID][0]
	_, err := svc.CreateLine(context.Background(), "PERAAAAA2", model.PayrollLineInput{CrewPublicID: fx.crewB.PublicID, BankAccountID: foreign.ID})
	fields := appErrorFields(t, err, 422, "validation_failed")
	assert.Equal(t, "Choose one of this crew member's bank accounts.", fields["bank_account_id"])

	_, err = svc.CreateLine(context.Background(), "PERAAAAA2", model.PayrollLineInput{CrewPublicID: "NOBODY999", BankAccountID: foreign.ID})
	fields = appErrorFields(t, err, 422, "validation_failed")
	assert.Equal(t, "The selected crew member does not exist.", fields["crew_id"])

	store.period.TransferDate = nil
	_, err = svc.CreateLine(context.Background(), "PERAAAAA2", model.PayrollLineInput{CrewPublicID: fx.crewA.PublicID, BankAccountID: foreign.ID})
	fields = appErrorFields(t, err, 422, "validation_failed")
	assert.Equal(t, "A date is required when the period has no transfer date.", fields["transfer_date"])

	_, err = svc.CreateLine(context.Background(), "UNKNOWN99", model.PayrollLineInput{})
	requireAppError(t, err, 404, "not_found")
}

func TestPayrollService_UpdateLineKeepsSnapshotsUnlessCrewOrAccountChanges(t *testing.T) {
	svc, store, fx := newPayrollServiceFixture()
	account := store.accounts[fx.crewA.ID][0]
	existing := model.PayrollLine{
		ID: uuid.New(), PeriodID: store.period.ID, LineNo: 4, CrewID: fx.crewA.ID, BankAccountID: &account.ID,
		CrewName: "BUDI SANTOSO (old)", RankCode: "3E", VesselName: "MV. OLD", BankName: account.BankName, AccountNumber: "OLD-SNAPSHOT",
		TransferDate: transferDay(),
	}
	store.lines[existing.ID] = existing
	date := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)

	l, err := svc.UpdateLine(context.Background(), existing.ID, model.PayrollLineInput{
		CrewPublicID: fx.crewA.PublicID, BankAccountID: account.ID, AmountIDR: decimal.RequireFromString("9"), TransferDate: &date,
	})
	require.NoError(t, err)
	assert.Equal(t, "BUDI SANTOSO (old)", l.CrewName)
	assert.Equal(t, "3E", l.RankCode)
	assert.Equal(t, "OLD-SNAPSHOT", l.AccountNumber)
	assert.Equal(t, 4, l.LineNo)
	assert.Equal(t, date, l.TransferDate)

	other := store.accounts[fx.crewA.ID][1]
	l, err = svc.UpdateLine(context.Background(), existing.ID, model.PayrollLineInput{CrewPublicID: fx.crewA.PublicID, BankAccountID: other.ID})
	require.NoError(t, err)
	assert.Equal(t, "0123456789", l.AccountNumber)
	assert.Equal(t, "BUDI SANTOSO (old)", l.CrewName)

	store.accounts[fx.crewB.ID] = []model.BankAccount{{ID: uuid.New(), CrewID: fx.crewB.ID, BankName: "BCA", AccountNumber: "555"}}
	l, err = svc.UpdateLine(context.Background(), existing.ID, model.PayrollLineInput{CrewPublicID: fx.crewB.PublicID, BankAccountID: store.accounts[fx.crewB.ID][0].ID})
	require.NoError(t, err)
	assert.Equal(t, "Candra Wijaya", l.CrewName)
	assert.Empty(t, l.RankCode)
	assert.Equal(t, "555", l.AccountNumber)

	_, err = svc.UpdateLine(context.Background(), uuid.New(), model.PayrollLineInput{})
	requireAppError(t, err, 404, "not_found")
}

func TestPayrollService_LinesReturnsPageAndTotals(t *testing.T) {
	svc, store, _ := newPayrollServiceFixture()
	f := model.PayrollLineFilter{Banks: []string{"MANDIRI"}, Q: "santoso"}
	page, totals, err := svc.Lines(context.Background(), "PERAAAAA2", f, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 3, page.Total)
	assert.Equal(t, 3, totals.Count)
	assert.Equal(t, f, store.pageFilter)
	assert.Equal(t, f, store.totalFilter)

	_, _, err = svc.Lines(context.Background(), "UNKNOWN99", f, 1, 20)
	requireAppError(t, err, 404, "not_found")
}

func TestPayrollService_CreatePeriodDrawsPublicID(t *testing.T) {
	svc, _, _ := newPayrollServiceFixture()
	p, err := svc.CreatePeriod(context.Background(), model.PayrollPeriod{Title: "Salary October 2026"})
	require.NoError(t, err)
	assert.Len(t, p.PublicID, 9)
}
