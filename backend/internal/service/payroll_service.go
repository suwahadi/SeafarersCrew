package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/publicid"
)

// PayrollStore is the persistence the payroll register needs.
type PayrollStore interface {
	CreatePeriod(ctx context.Context, p model.PayrollPeriod, newID func() (string, error), attempts int) (model.PayrollPeriod, error)
	GetPeriod(ctx context.Context, publicID string) (model.PayrollPeriod, error)
	PeriodByID(ctx context.Context, id uuid.UUID) (model.PayrollPeriod, error)
	ListPeriods(ctx context.Context, q string, page, perPage int) (repository.Page[model.PayrollPeriod], error)
	UpdatePeriod(ctx context.Context, p model.PayrollPeriod) (model.PayrollPeriod, error)
	DeletePeriod(ctx context.Context, id uuid.UUID) error
	ListLines(ctx context.Context, periodID uuid.UUID, f model.PayrollLineFilter, page, perPage int) (repository.Page[model.PayrollLine], error)
	LineTotals(ctx context.Context, periodID uuid.UUID, f model.PayrollLineFilter) (model.PayrollTotals, error)
	Facets(ctx context.Context, periodID uuid.UUID) (model.PayrollFacets, error)
	GetLine(ctx context.Context, id uuid.UUID) (model.PayrollLine, error)
	SaveLine(ctx context.Context, l model.PayrollLine) (model.PayrollLine, error)
	DeleteLine(ctx context.Context, id uuid.UUID) error
	CrewByPublicID(ctx context.Context, publicID string) (model.PayrollCrew, error)
	BankAccountsByCrew(ctx context.Context, crewIDs []uuid.UUID) (map[uuid.UUID][]model.BankAccount, error)
	CoveringContracts(ctx context.Context, crewIDs []uuid.UUID, start, end time.Time) (map[uuid.UUID]model.PayrollPlacement, error)
}

// PayrollService manages payroll periods and their transfer lines.
type PayrollService struct {
	store PayrollStore
}

// NewPayrollService builds a PayrollService.
func NewPayrollService(store PayrollStore) *PayrollService {
	return &PayrollService{store: store}
}

func payrollNotFound(err error, what string) error {
	if errors.Is(err, repository.ErrNotFound) {
		return apperr.NotFound(what)
	}
	return err
}

// CreatePeriod adds a period.
func (s *PayrollService) CreatePeriod(ctx context.Context, in model.PayrollPeriod) (model.PayrollPeriod, error) {
	return s.store.CreatePeriod(ctx, in, publicid.New, publicid.MaxAttempts)
}

// GetPeriod returns a period with its totals.
func (s *PayrollService) GetPeriod(ctx context.Context, publicID string) (model.PayrollPeriod, error) {
	p, err := s.store.GetPeriod(ctx, publicID)
	return p, payrollNotFound(err, "Payroll period")
}

// ListPeriods returns a page of periods, newest first.
func (s *PayrollService) ListPeriods(ctx context.Context, q string, page, perPage int) (repository.Page[model.PayrollPeriod], error) {
	return s.store.ListPeriods(ctx, q, page, perPage)
}

// UpdatePeriod saves the title, dates and transfer date of a period.
func (s *PayrollService) UpdatePeriod(ctx context.Context, publicID string, in model.PayrollPeriod) (model.PayrollPeriod, error) {
	cur, err := s.GetPeriod(ctx, publicID)
	if err != nil {
		return cur, err
	}
	in.ID, in.PublicID = cur.ID, cur.PublicID
	p, err := s.store.UpdatePeriod(ctx, in)
	return p, payrollNotFound(err, "Payroll period")
}

// DeletePeriod removes a period with all its lines.
func (s *PayrollService) DeletePeriod(ctx context.Context, publicID string) error {
	cur, err := s.GetPeriod(ctx, publicID)
	if err != nil {
		return err
	}
	return payrollNotFound(s.store.DeletePeriod(ctx, cur.ID), "Payroll period")
}

// Lines returns one page of a period's lines and the totals of every line matching f.
func (s *PayrollService) Lines(ctx context.Context, publicID string, f model.PayrollLineFilter, page, perPage int) (repository.Page[model.PayrollLine], model.PayrollTotals, error) {
	var (
		lines  repository.Page[model.PayrollLine]
		totals model.PayrollTotals
	)
	p, err := s.GetPeriod(ctx, publicID)
	if err != nil {
		return lines, totals, err
	}
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		lines, err = s.store.ListLines(gctx, p.ID, f, page, perPage)
		return err
	})
	g.Go(func() error {
		var err error
		totals, err = s.store.LineTotals(gctx, p.ID, f)
		return err
	})
	return lines, totals, g.Wait()
}

// Facets lists the banks, vessels and ranks used in a period.
func (s *PayrollService) Facets(ctx context.Context, publicID string) (model.PayrollFacets, error) {
	p, err := s.GetPeriod(ctx, publicID)
	if err != nil {
		return model.PayrollFacets{}, err
	}
	return s.store.Facets(ctx, p.ID)
}

// CreateLine adds a line at the end of a period.
func (s *PayrollService) CreateLine(ctx context.Context, periodPublicID string, in model.PayrollLineInput) (model.PayrollLine, error) {
	p, err := s.GetPeriod(ctx, periodPublicID)
	if err != nil {
		return model.PayrollLine{}, err
	}
	l, err := s.fill(ctx, p, model.PayrollLine{PeriodID: p.ID}, in)
	if err != nil {
		return l, err
	}
	return s.store.SaveLine(ctx, l)
}

// UpdateLine edits a line; crew and bank snapshots refresh only when the crew member or account changes.
func (s *PayrollService) UpdateLine(ctx context.Context, id uuid.UUID, in model.PayrollLineInput) (model.PayrollLine, error) {
	cur, err := s.store.GetLine(ctx, id)
	if err != nil {
		return cur, payrollNotFound(err, "Payroll line")
	}
	p, err := s.store.PeriodByID(ctx, cur.PeriodID)
	if err != nil {
		return cur, payrollNotFound(err, "Payroll period")
	}
	l, err := s.fill(ctx, p, cur, in)
	if err != nil {
		return l, err
	}
	saved, err := s.store.SaveLine(ctx, l)
	return saved, payrollNotFound(err, "Payroll line")
}

// DeleteLine removes a line.
func (s *PayrollService) DeleteLine(ctx context.Context, id uuid.UUID) error {
	return payrollNotFound(s.store.DeleteLine(ctx, id), "Payroll line")
}

func (s *PayrollService) fill(ctx context.Context, p model.PayrollPeriod, l model.PayrollLine, in model.PayrollLineInput) (model.PayrollLine, error) {
	crew, err := s.store.CrewByPublicID(ctx, in.CrewPublicID)
	if errors.Is(err, repository.ErrNotFound) {
		return l, apperr.Validation(map[string]string{"crew_id": "The selected crew member does not exist."})
	}
	if err != nil {
		return l, err
	}
	accounts, err := s.store.BankAccountsByCrew(ctx, []uuid.UUID{crew.ID})
	if err != nil {
		return l, err
	}
	var account *model.BankAccount
	for i, a := range accounts[crew.ID] {
		if a.ID == in.BankAccountID {
			account = &accounts[crew.ID][i]
		}
	}
	if account == nil {
		return l, apperr.Validation(map[string]string{"bank_account_id": "Choose one of this crew member's bank accounts."})
	}
	date := in.TransferDate
	if date == nil {
		date = p.TransferDate
	}
	if date == nil {
		return l, apperr.Validation(map[string]string{"transfer_date": "A date is required when the period has no transfer date."})
	}
	crewChanged := l.ID == uuid.Nil || l.CrewID != crew.ID
	accountChanged := crewChanged || l.BankAccountID == nil || *l.BankAccountID != account.ID
	if crewChanged {
		placements, err := s.store.CoveringContracts(ctx, []uuid.UUID{crew.ID}, p.PeriodStart, p.PeriodEnd)
		if err != nil {
			return l, err
		}
		placement := placements[crew.ID]
		l.CrewID, l.CrewName, l.EmployeeNo = crew.ID, crew.FullName, crew.EmployeeNo
		l.RankCode, l.VesselName = placement.RankCode, placement.VesselName
	}
	if accountChanged {
		id := account.ID
		l.BankAccountID = &id
		l.BankName, l.AccountName, l.AccountNumber = account.BankName, account.AccountName, account.AccountNumber
	}
	l.AmountUSD, l.AmountIDR, l.TransferDate, l.Remark = in.AmountUSD, in.AmountIDR, *date, in.Remark
	return l, nil
}
