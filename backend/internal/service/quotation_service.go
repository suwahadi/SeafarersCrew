package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/publicid"
)

// QuotationStore persists quotations.
type QuotationStore interface {
	List(ctx context.Context, f model.QuotationFilter) (repository.Page[model.Quotation], error)
	GetByPublicID(ctx context.Context, publicID string) (model.Quotation, error)
	GetByID(ctx context.Context, id uuid.UUID) (model.Quotation, error)
	Create(ctx context.Context, q model.Quotation) (uuid.UUID, error)
	UpdateDraft(ctx context.Context, q model.Quotation) error
	SetStatus(ctx context.Context, id uuid.UUID, from, to string) (bool, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	ExpireOverdue(ctx context.Context, today time.Time) (int64, error)
	CheckRefs(ctx context.Context, clientID uuid.UUID, vesselID *uuid.UUID, currency string, rankIDs []uuid.UUID) (repository.QuotationRefs, error)
}

// QuotationService manages quotations (PRD 6.6).
type QuotationService struct {
	store QuotationStore
	newID func() (string, error)
}

// NewQuotationService builds a QuotationService.
func NewQuotationService(store QuotationStore) *QuotationService {
	return &QuotationService{store: store, newID: publicid.New}
}

var quotationTransitions = map[string][]string{
	model.QuotationDraft: {model.QuotationSent},
	model.QuotationSent:  {model.QuotationAccepted, model.QuotationRejected, model.QuotationExpired},
}

// CanTransition reports whether a quotation may move from one status to another by hand.
func CanTransition(from, to string) bool {
	return slices.Contains(quotationTransitions[from], to)
}

// ComputeQuotation sets line amounts, tax = (subtotal − discount) × tax% and grand total, rounded to the currency decimals.
func ComputeQuotation(q *model.Quotation, decimals int32) error {
	subtotal := decimal.Zero
	for i := range q.Lines {
		l := &q.Lines[i]
		if l.LineType == model.LineDailyRate && l.DailyRate != nil && l.CrewCount != nil && l.Days != nil {
			l.Amount = l.DailyRate.Mul(decimal.NewFromInt(int64(*l.CrewCount))).Mul(decimal.NewFromInt(int64(*l.Days)))
		}
		l.Amount = l.Amount.Round(decimals)
		subtotal = subtotal.Add(l.Amount)
	}
	q.Subtotal = subtotal
	q.DiscountAmount = q.DiscountAmount.Round(decimals)
	if q.DiscountAmount.GreaterThan(q.Subtotal) {
		return apperr.Validation(map[string]string{"discount_amount": "The discount may not exceed the subtotal."})
	}
	taxable := q.Subtotal.Sub(q.DiscountAmount)
	q.TaxAmount = taxable.Mul(q.TaxPercent).Div(decimal.NewFromInt(100)).Round(decimals)
	q.GrandTotal = taxable.Add(q.TaxAmount)
	if q.GrandTotal.GreaterThanOrEqual(decimal.New(1, 16)) {
		return apperr.Validation(map[string]string{"lines": "The total is too large."})
	}
	return nil
}

// List returns one page of quotations.
func (s *QuotationService) List(ctx context.Context, f model.QuotationFilter) (repository.Page[model.Quotation], error) {
	return s.store.List(ctx, f)
}

// Get returns a quotation by public ID.
func (s *QuotationService) Get(ctx context.Context, publicID string) (model.Quotation, error) {
	if !publicid.Valid(publicID) {
		return model.Quotation{}, apperr.NotFound("Quotation")
	}
	q, err := s.store.GetByPublicID(ctx, publicID)
	if errors.Is(err, repository.ErrNotFound) {
		return q, apperr.NotFound("Quotation")
	}
	return q, err
}

// Create validates references, computes totals and stores a draft quotation with a unique public ID.
func (s *QuotationService) Create(ctx context.Context, q model.Quotation) (model.Quotation, error) {
	if err := s.prepare(ctx, &q); err != nil {
		return model.Quotation{}, err
	}
	for range publicid.MaxAttempts {
		pid, err := s.newID()
		if err != nil {
			return model.Quotation{}, err
		}
		q.PublicID = pid
		id, err := s.store.Create(ctx, q)
		if postgres.IsUniqueViolation(err, "quotations_public_id_key") {
			continue
		}
		if err != nil {
			return model.Quotation{}, err
		}
		return s.store.GetByID(ctx, id)
	}
	return model.Quotation{}, fmt.Errorf("create quotation: no unique public id after %d attempts", publicid.MaxAttempts)
}

// Update replaces a draft quotation.
func (s *QuotationService) Update(ctx context.Context, publicID string, q model.Quotation) (model.Quotation, error) {
	cur, err := s.Get(ctx, publicID)
	if err != nil {
		return model.Quotation{}, err
	}
	if cur.Status != model.QuotationDraft {
		return model.Quotation{}, apperr.Conflict("quotation_locked", "Only draft quotations can be edited.")
	}
	if err := s.prepare(ctx, &q); err != nil {
		return model.Quotation{}, err
	}
	q.ID = cur.ID
	if err := s.store.UpdateDraft(ctx, q); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Quotation{}, apperr.Conflict("quotation_locked", "Only draft quotations can be edited.")
		}
		return model.Quotation{}, err
	}
	return s.store.GetByID(ctx, cur.ID)
}

// ChangeStatus applies a manual status transition.
func (s *QuotationService) ChangeStatus(ctx context.Context, publicID, to string) (model.Quotation, error) {
	cur, err := s.Get(ctx, publicID)
	if err != nil {
		return model.Quotation{}, err
	}
	if !CanTransition(cur.Status, to) {
		return model.Quotation{}, apperr.Conflict("invalid_transition", fmt.Sprintf("A %s quotation cannot become %s.", cur.Status, to))
	}
	ok, err := s.store.SetStatus(ctx, cur.ID, cur.Status, to)
	if err != nil {
		return model.Quotation{}, err
	}
	if !ok {
		return model.Quotation{}, apperr.Conflict("status_changed", "The quotation changed meanwhile; reload and try again.")
	}
	return s.store.GetByID(ctx, cur.ID)
}

// Delete soft-deletes a quotation.
func (s *QuotationService) Delete(ctx context.Context, publicID string) error {
	cur, err := s.Get(ctx, publicID)
	if err != nil {
		return err
	}
	return s.store.SoftDelete(ctx, cur.ID)
}

// ExpireQuotations marks draft and sent quotations past their valid-until date as expired.
func (s *QuotationService) ExpireQuotations(ctx context.Context, today time.Time) (int64, error) {
	return s.store.ExpireOverdue(ctx, today)
}

func (s *QuotationService) prepare(ctx context.Context, q *model.Quotation) error {
	var rankIDs []uuid.UUID
	for _, l := range q.Lines {
		if l.RankID != nil {
			rankIDs = append(rankIDs, *l.RankID)
		}
	}
	refs, err := s.store.CheckRefs(ctx, q.ClientCompanyID, q.VesselID, q.CurrencyCode, rankIDs)
	if err != nil {
		return err
	}
	fields := map[string]string{}
	switch {
	case refs.ClientType == "":
		fields["client_company_id"] = "The selected client does not exist."
	case refs.ClientType == "company":
		fields["client_company_id"] = "The client must be a principal."
	case !refs.ClientActive:
		fields["client_company_id"] = "The selected client is inactive."
	}
	if q.VesselID != nil && !refs.VesselExists {
		fields["vessel_id"] = "The selected vessel does not exist."
	}
	if !refs.CurrencyActive {
		fields["currency_code"] = "The selected currency does not exist or is inactive."
	}
	if refs.MissingRanks > 0 {
		fields["lines"] = "A selected rank does not exist."
	}
	if len(fields) > 0 {
		return apperr.Validation(fields)
	}
	q.CurrencyDecimal = refs.CurrencyDecimal
	return ComputeQuotation(q, refs.CurrencyDecimal)
}
