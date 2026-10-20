package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// QuotationRepository stores quotations and their lines.
type QuotationRepository struct {
	pool *pgxpool.Pool
}

// NewQuotationRepository returns a QuotationRepository.
func NewQuotationRepository(pool *pgxpool.Pool) *QuotationRepository {
	return &QuotationRepository{pool: pool}
}

const quotationSelect = `SELECT q.id, q.public_id, q.client_company_id, c.name, q.vessel_id, coalesce(v.name, ''), q.currency_code,
	cur.decimals, q.quotation_date, q.valid_until, q.tax_percent, q.discount_amount, q.terms, q.notes, q.status, q.subtotal,
	q.tax_amount, q.grand_total, q.created_at, q.updated_at
FROM quotations q
JOIN companies c ON c.id = q.client_company_id
JOIN currencies cur ON cur.code = q.currency_code
LEFT JOIN vessels v ON v.id = q.vessel_id`

func scanQuotation(row pgx.Row) (model.Quotation, error) {
	var q model.Quotation
	var decimals int16
	err := row.Scan(&q.ID, &q.PublicID, &q.ClientCompanyID, &q.ClientName, &q.VesselID, &q.VesselName, &q.CurrencyCode,
		&decimals, &q.QuotationDate, &q.ValidUntil, &q.TaxPercent, &q.DiscountAmount, &q.Terms, &q.Notes, &q.Status, &q.Subtotal,
		&q.TaxAmount, &q.GrandTotal, &q.CreatedAt, &q.UpdatedAt)
	q.CurrencyDecimal = int32(decimals)
	return q, notFound(err)
}

func quotationWhere(f model.QuotationFilter) *Where {
	w := &Where{}
	w.Add("q.deleted_at IS NULL")
	if f.Q != "" {
		w.Add("(q.public_id ILIKE %s OR c.name ILIKE %s OR coalesce(v.name, '') ILIKE %s)", LikePattern(f.Q), LikePattern(f.Q), LikePattern(f.Q))
	}
	if len(f.Statuses) > 0 {
		w.Add("q.status = ANY(%s)", f.Statuses)
	}
	if len(f.ClientIDs) > 0 {
		w.Add("q.client_company_id = ANY(%s)", f.ClientIDs)
	}
	if len(f.Currencies) > 0 {
		w.Add("q.currency_code = ANY(%s)", f.Currencies)
	}
	if f.DateFrom != nil {
		w.Add("q.quotation_date >= %s", *f.DateFrom)
	}
	if f.DateTo != nil {
		w.Add("q.quotation_date <= %s", *f.DateTo)
	}
	return w
}

// List returns one page of quotations without lines.
func (r *QuotationRepository) List(ctx context.Context, f model.QuotationFilter) (Page[model.Quotation], error) {
	var page Page[model.Quotation]
	from := ` FROM quotations q JOIN companies c ON c.id = q.client_company_id LEFT JOIN vessels v ON v.id = q.vessel_id`
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		w := quotationWhere(f)
		return r.pool.QueryRow(gctx, `SELECT count(*)`+from+w.SQL(), w.Args()...).Scan(&page.Total)
	})
	g.Go(func() error {
		w := quotationWhere(f)
		order := OrderWithTie(f.Sort, f.Desc, map[string]string{
			"id": "q.public_id", "client": "c.name", "vessel": "v.name", "quotation_date": "q.quotation_date",
			"valid_until": "q.valid_until", "status": "q.status", "currency": "q.currency_code", "grand_total": "q.grand_total",
			"created_at": "q.created_at",
		}, "q.quotation_date DESC, q.id DESC", "q.id")
		lim, off := w.Arg(f.Limit), w.Arg(f.Offset)
		rows, err := r.pool.Query(gctx, quotationSelect+w.SQL()+order+` LIMIT `+lim+` OFFSET `+off, w.Args()...)
		if err != nil {
			return err
		}
		page.Items, err = collect(rows, scanQuotation, "quotations")
		return err
	})
	if err := g.Wait(); err != nil {
		return page, fmt.Errorf("list quotations: %w", err)
	}
	return page, nil
}

// GetByPublicID returns a quotation with its lines.
func (r *QuotationRepository) GetByPublicID(ctx context.Context, publicID string) (model.Quotation, error) {
	return r.get(ctx, `q.public_id = $1`, publicID)
}

// GetByID returns a quotation with its lines.
func (r *QuotationRepository) GetByID(ctx context.Context, id uuid.UUID) (model.Quotation, error) {
	return r.get(ctx, `q.id = $1`, id)
}

func (r *QuotationRepository) get(ctx context.Context, cond string, arg any) (model.Quotation, error) {
	q, err := scanQuotation(r.pool.QueryRow(ctx, quotationSelect+` WHERE `+cond+` AND q.deleted_at IS NULL`, arg))
	if err != nil {
		return q, fmt.Errorf("get quotation: %w", err)
	}
	rows, err := r.pool.Query(ctx, `SELECT l.id, l.line_type, l.rank_id, coalesce(rk.name, ''), l.description, l.crew_count,
		l.daily_rate, l.days, l.amount, l.sort_order
		FROM quotation_lines l LEFT JOIN ranks rk ON rk.id = l.rank_id
		WHERE l.quotation_id = $1 ORDER BY l.sort_order, l.created_at`, q.ID)
	if err != nil {
		return q, fmt.Errorf("list quotation lines: %w", err)
	}
	q.Lines, err = collect(rows, func(row pgx.Row) (model.QuotationLine, error) {
		var l model.QuotationLine
		err := row.Scan(&l.ID, &l.LineType, &l.RankID, &l.RankName, &l.Description, &l.CrewCount, &l.DailyRate, &l.Days, &l.Amount, &l.SortOrder)
		return l, err
	}, "quotation lines")
	return q, err
}

// Create inserts a quotation with its lines and returns its ID.
func (r *QuotationRepository) Create(ctx context.Context, q model.Quotation) (uuid.UUID, error) {
	var id uuid.UUID
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `INSERT INTO quotations (public_id, client_company_id, vessel_id, currency_code, quotation_date,
			valid_until, tax_percent, discount_amount, terms, notes, status, subtotal, tax_amount, grand_total)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'draft', $11, $12, $13) RETURNING id`,
			q.PublicID, q.ClientCompanyID, q.VesselID, q.CurrencyCode, q.QuotationDate, q.ValidUntil, q.TaxPercent,
			q.DiscountAmount, q.Terms, q.Notes, q.Subtotal, q.TaxAmount, q.GrandTotal).Scan(&id)
		if err != nil {
			return err
		}
		return insertQuotationLines(ctx, tx, id, q.Lines)
	})
	return id, err
}

// UpdateDraft replaces a draft quotation and its lines; ErrNotFound when it is missing or no longer a draft.
func (r *QuotationRepository) UpdateDraft(ctx context.Context, q model.Quotation) error {
	return InTx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE quotations SET client_company_id = $2, vessel_id = $3, currency_code = $4, quotation_date = $5,
			valid_until = $6, tax_percent = $7, discount_amount = $8, terms = $9, notes = $10, subtotal = $11, tax_amount = $12,
			grand_total = $13 WHERE id = $1 AND status = 'draft' AND deleted_at IS NULL`,
			q.ID, q.ClientCompanyID, q.VesselID, q.CurrencyCode, q.QuotationDate, q.ValidUntil, q.TaxPercent, q.DiscountAmount,
			q.Terms, q.Notes, q.Subtotal, q.TaxAmount, q.GrandTotal)
		if err := affectedOne(tag, err, "update quotation"); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM quotation_lines WHERE quotation_id = $1`, q.ID); err != nil {
			return fmt.Errorf("replace quotation lines: %w", err)
		}
		return insertQuotationLines(ctx, tx, q.ID, q.Lines)
	})
}

func insertQuotationLines(ctx context.Context, tx pgx.Tx, quotationID uuid.UUID, lines []model.QuotationLine) error {
	batch := &pgx.Batch{}
	for i, l := range lines {
		batch.Queue(`INSERT INTO quotation_lines (quotation_id, line_type, rank_id, description, crew_count, daily_rate, days, amount, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			quotationID, l.LineType, l.RankID, l.Description, l.CrewCount, l.DailyRate, l.Days, l.Amount, (i+1)*10)
	}
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("insert quotation lines: %w", err)
	}
	return nil
}

// SetStatus moves a quotation from one status to another and reports whether it changed.
func (r *QuotationRepository) SetStatus(ctx context.Context, id uuid.UUID, from, to string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE quotations SET status = $3 WHERE id = $1 AND status = $2 AND deleted_at IS NULL`, id, from, to)
	if err != nil {
		return false, fmt.Errorf("set quotation status: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// SoftDelete marks a quotation deleted.
func (r *QuotationRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE quotations SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return affectedOne(tag, err, "delete quotation")
}

// ExpireOverdue marks draft and sent quotations whose validity ended before today as expired.
func (r *QuotationRepository) ExpireOverdue(ctx context.Context, today time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE quotations SET status = 'expired'
		WHERE status IN ('draft', 'sent') AND valid_until < $1::date AND deleted_at IS NULL`, today)
	if err != nil {
		return 0, fmt.Errorf("expire quotations: %w", err)
	}
	return tag.RowsAffected(), nil
}

// QuotationRefs holds the reference checks of a quotation.
type QuotationRefs struct {
	ClientType      string
	ClientActive    bool
	VesselExists    bool
	CurrencyActive  bool
	CurrencyDecimal int32
	MissingRanks    int
}

// CheckRefs loads what is needed to validate a quotation's client, vessel, currency and line ranks.
func (r *QuotationRepository) CheckRefs(ctx context.Context, clientID uuid.UUID, vesselID *uuid.UUID, currency string, rankIDs []uuid.UUID) (QuotationRefs, error) {
	var refs QuotationRefs
	var clientType, vessel *string
	var decimals *int16
	err := r.pool.QueryRow(ctx, `SELECT
		(SELECT type FROM companies WHERE id = $1 AND deleted_at IS NULL),
		coalesce((SELECT is_active FROM companies WHERE id = $1 AND deleted_at IS NULL), false),
		(SELECT id::text FROM vessels WHERE id = $2 AND deleted_at IS NULL),
		(SELECT decimals FROM currencies WHERE code = $3 AND is_active AND deleted_at IS NULL),
		(SELECT count(*) FROM unnest($4::uuid[]) x(id) WHERE NOT EXISTS (SELECT 1 FROM ranks r WHERE r.id = x.id AND r.deleted_at IS NULL))`,
		clientID, vesselID, currency, rankIDs).Scan(&clientType, &refs.ClientActive, &vessel, &decimals, &refs.MissingRanks)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return refs, fmt.Errorf("check quotation references: %w", err)
	}
	if clientType != nil {
		refs.ClientType = *clientType
	}
	refs.VesselExists = vessel != nil
	if decimals != nil {
		refs.CurrencyActive = true
		refs.CurrencyDecimal = int32(*decimals)
	}
	return refs, nil
}
