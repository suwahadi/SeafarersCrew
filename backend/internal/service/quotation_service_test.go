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
)

func dptr(s string) *decimal.Decimal { d := dec(s); return &d }

func iptr(n int) *int { return &n }

func dailyLine(rate string, crew, days int) model.QuotationLine {
	r := uuid.New()
	return model.QuotationLine{LineType: model.LineDailyRate, RankID: &r, DailyRate: dptr(rate), CrewCount: iptr(crew), Days: iptr(days)}
}

func lumpLine(desc, amount string) model.QuotationLine {
	return model.QuotationLine{LineType: model.LineLumpSum, Description: desc, Amount: dec(amount)}
}

func TestComputeQuotation(t *testing.T) {
	cases := []struct {
		name     string
		lines    []model.QuotationLine
		decimals int32
		tax      string
		discount string
		subtotal string
		taxAmt   string
		grand    string
	}{
		{"daily rate only", []model.QuotationLine{dailyLine("250.50", 3, 30)}, 2, "0", "0", "22545", "0", "22545"},
		{"lump sum only", []model.QuotationLine{lumpLine("Mobilisation", "1500"), lumpLine("Visa", "275.25")}, 2, "11", "0", "1775.25", "195.28", "1970.53"},
		{"both with tax and discount", []model.QuotationLine{dailyLine("300", 2, 10), lumpLine("Travel", "1000")}, 2, "10", "500", "7000", "650", "7150"},
		{"idr rounds to whole rupiah", []model.QuotationLine{dailyLine("1500000.50", 1, 3)}, 0, "11", "0.4", "4500002", "495000", "4995002"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := model.Quotation{Lines: tc.lines, TaxPercent: dec(tc.tax), DiscountAmount: dec(tc.discount)}
			require.NoError(t, ComputeQuotation(&q, tc.decimals))
			assert.True(t, dec(tc.subtotal).Equal(q.Subtotal), "subtotal %s", q.Subtotal)
			assert.True(t, dec(tc.taxAmt).Equal(q.TaxAmount), "tax %s", q.TaxAmount)
			assert.True(t, dec(tc.grand).Equal(q.GrandTotal), "grand %s", q.GrandTotal)
			sum := decimal.Zero
			for _, l := range q.Lines {
				sum = sum.Add(l.Amount)
			}
			assert.True(t, sum.Equal(q.Subtotal))
		})
	}
}

func TestComputeQuotationRejectsDiscountAboveSubtotal(t *testing.T) {
	q := model.Quotation{Lines: []model.QuotationLine{lumpLine("A", "100")}, DiscountAmount: dec("100.01")}
	e := requireAppErr(t, ComputeQuotation(&q, 2), "validation_failed")
	assert.Contains(t, e.Fields, "discount_amount")
}

func TestQuotationTransitions(t *testing.T) {
	allowed := map[string][]string{
		model.QuotationDraft: {model.QuotationSent},
		model.QuotationSent:  {model.QuotationAccepted, model.QuotationRejected, model.QuotationExpired},
	}
	all := []string{model.QuotationDraft, model.QuotationSent, model.QuotationAccepted, model.QuotationRejected, model.QuotationExpired}
	for _, from := range all {
		for _, to := range all {
			want := false
			for _, a := range allowed[from] {
				want = want || a == to
			}
			assert.Equal(t, want, CanTransition(from, to), "%s -> %s", from, to)
		}
	}
}

type memQuotations struct {
	refs       repository.QuotationRefs
	byID       map[uuid.UUID]model.Quotation
	collisions int
	expired    time.Time
}

func newMemQuotations() *memQuotations {
	return &memQuotations{byID: map[uuid.UUID]model.Quotation{}, refs: repository.QuotationRefs{
		ClientType: "principal", ClientActive: true, VesselExists: true, CurrencyActive: true, CurrencyDecimal: 2,
	}}
}

func (m *memQuotations) List(context.Context, model.QuotationFilter) (repository.Page[model.Quotation], error) {
	return repository.Page[model.Quotation]{}, nil
}

func (m *memQuotations) GetByPublicID(_ context.Context, pid string) (model.Quotation, error) {
	for _, q := range m.byID {
		if q.PublicID == pid {
			return q, nil
		}
	}
	return model.Quotation{}, repository.ErrNotFound
}

func (m *memQuotations) GetByID(_ context.Context, id uuid.UUID) (model.Quotation, error) {
	return m.byID[id], nil
}

func (m *memQuotations) Create(_ context.Context, q model.Quotation) (uuid.UUID, error) {
	if m.collisions > 0 {
		m.collisions--
		return uuid.Nil, &pgconn.PgError{Code: "23505", ConstraintName: "quotations_public_id_key"}
	}
	q.ID, q.Status = uuid.New(), model.QuotationDraft
	m.byID[q.ID] = q
	return q.ID, nil
}

func (m *memQuotations) UpdateDraft(_ context.Context, q model.Quotation) error {
	q.Status = m.byID[q.ID].Status
	q.PublicID = m.byID[q.ID].PublicID
	m.byID[q.ID] = q
	return nil
}

func (m *memQuotations) SetStatus(_ context.Context, id uuid.UUID, from, to string) (bool, error) {
	q := m.byID[id]
	if q.Status != from {
		return false, nil
	}
	q.Status = to
	m.byID[id] = q
	return true, nil
}

func (m *memQuotations) SoftDelete(_ context.Context, id uuid.UUID) error {
	delete(m.byID, id)
	return nil
}

func (m *memQuotations) ExpireOverdue(_ context.Context, today time.Time) (int64, error) {
	m.expired = today
	var n int64
	for id, q := range m.byID {
		if (q.Status == model.QuotationDraft || q.Status == model.QuotationSent) && q.ValidUntil.Before(today) {
			q.Status = model.QuotationExpired
			m.byID[id] = q
			n++
		}
	}
	return n, nil
}

func (m *memQuotations) CheckRefs(context.Context, uuid.UUID, *uuid.UUID, string, []uuid.UUID) (repository.QuotationRefs, error) {
	return m.refs, nil
}

func newQuotation() model.Quotation {
	return model.Quotation{ClientCompanyID: uuid.New(), CurrencyCode: "USD", QuotationDate: date(2026, 10, 1), ValidUntil: date(2026, 10, 31),
		TaxPercent: dec("10"), Lines: []model.QuotationLine{dailyLine("100", 2, 5)}}
}

func TestQuotationCreateComputesTotalsAndRetriesPublicID(t *testing.T) {
	store := newMemQuotations()
	store.collisions = 2
	svc := NewQuotationService(store)
	q, err := svc.Create(context.Background(), newQuotation())
	require.NoError(t, err)
	assert.Regexp(t, `^[2-9A-HJKMNP-Z]{9}$`, q.PublicID)
	assert.True(t, dec("1100").Equal(q.GrandTotal))
	assert.Equal(t, model.QuotationDraft, q.Status)
}

func TestQuotationReferenceValidation(t *testing.T) {
	cases := []struct {
		name  string
		refs  repository.QuotationRefs
		field string
	}{
		{"missing client", repository.QuotationRefs{CurrencyActive: true}, "client_company_id"},
		{"client is not a principal", repository.QuotationRefs{ClientType: "company", ClientActive: true, CurrencyActive: true}, "client_company_id"},
		{"inactive currency", repository.QuotationRefs{ClientType: "both", ClientActive: true}, "currency_code"},
		{"missing vessel", repository.QuotationRefs{ClientType: "both", ClientActive: true, CurrencyActive: true}, "vessel_id"},
		{"missing rank", repository.QuotationRefs{ClientType: "both", ClientActive: true, CurrencyActive: true, VesselExists: true, MissingRanks: 1}, "lines"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newMemQuotations()
			store.refs = tc.refs
			q := newQuotation()
			v := uuid.New()
			q.VesselID = &v
			_, err := NewQuotationService(store).Create(context.Background(), q)
			e := requireAppErr(t, err, "validation_failed")
			assert.Contains(t, e.Fields, tc.field)
		})
	}
}

func TestOnlyDraftQuotationsCanBeEditedAndStatusesFollowTheFlow(t *testing.T) {
	ctx := context.Background()
	svc := NewQuotationService(newMemQuotations())
	q, err := svc.Create(ctx, newQuotation())
	require.NoError(t, err)

	_, err = svc.ChangeStatus(ctx, q.PublicID, model.QuotationAccepted)
	requireAppErr(t, err, "invalid_transition")

	edited := newQuotation()
	edited.Lines = []model.QuotationLine{lumpLine("Fee", "50")}
	updated, err := svc.Update(ctx, q.PublicID, edited)
	require.NoError(t, err)
	assert.True(t, dec("55").Equal(updated.GrandTotal))

	sent, err := svc.ChangeStatus(ctx, q.PublicID, model.QuotationSent)
	require.NoError(t, err)
	assert.Equal(t, model.QuotationSent, sent.Status)
	_, err = svc.Update(ctx, q.PublicID, edited)
	requireAppErr(t, err, "quotation_locked")

	accepted, err := svc.ChangeStatus(ctx, q.PublicID, model.QuotationAccepted)
	require.NoError(t, err)
	assert.Equal(t, model.QuotationAccepted, accepted.Status)
	_, err = svc.ChangeStatus(ctx, q.PublicID, model.QuotationRejected)
	requireAppErr(t, err, "invalid_transition")

	_, err = svc.Get(ctx, "lowercase")
	requireAppErr(t, err, "not_found")
}

func TestExpireQuotationsPastValidity(t *testing.T) {
	ctx := context.Background()
	store := newMemQuotations()
	svc := NewQuotationService(store)
	old, err := svc.Create(ctx, newQuotation())
	require.NoError(t, err)
	current := newQuotation()
	current.ValidUntil = date(2026, 12, 31)
	cur, err := svc.Create(ctx, current)
	require.NoError(t, err)

	n, err := svc.ExpireQuotations(ctx, date(2026, 11, 1))
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	assert.Equal(t, model.QuotationExpired, store.byID[old.ID].Status)
	assert.Equal(t, model.QuotationDraft, store.byID[cur.ID].Status)
}
