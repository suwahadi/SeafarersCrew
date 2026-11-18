package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
)

// PayrollRepository stores payroll periods and their transfer lines.
type PayrollRepository struct {
	pool *pgxpool.Pool
}

// NewPayrollRepository returns a PayrollRepository.
func NewPayrollRepository(pool *pgxpool.Pool) *PayrollRepository {
	return &PayrollRepository{pool: pool}
}

const payrollPeriodSelect = `SELECT p.id, p.public_id, p.title, p.period_start, p.period_end, p.transfer_date, p.created_at, p.updated_at,
	t.n, t.usd, t.idr
FROM payroll_periods p
CROSS JOIN LATERAL (SELECT count(*) AS n, coalesce(sum(amount_usd), 0) AS usd, coalesce(sum(amount_idr), 0) AS idr
	FROM payroll_lines WHERE period_id = p.id) t`

func scanPayrollPeriod(row pgx.Row) (model.PayrollPeriod, error) {
	var p model.PayrollPeriod
	err := row.Scan(&p.ID, &p.PublicID, &p.Title, &p.PeriodStart, &p.PeriodEnd, &p.TransferDate, &p.CreatedAt, &p.UpdatedAt,
		&p.LineCount, &p.TotalUSD, &p.TotalIDR)
	return p, notFound(err)
}

// CreatePeriod inserts a period, drawing a new public ID when one is already taken.
func (r *PayrollRepository) CreatePeriod(ctx context.Context, p model.PayrollPeriod, newID func() (string, error), attempts int) (model.PayrollPeriod, error) {
	for range attempts {
		pid, err := newID()
		if err != nil {
			return model.PayrollPeriod{}, err
		}
		_, err = r.pool.Exec(ctx, `INSERT INTO payroll_periods (public_id, title, period_start, period_end, transfer_date)
			VALUES ($1, $2, $3, $4, $5)`, pid, p.Title, p.PeriodStart, p.PeriodEnd, p.TransferDate)
		if err == nil {
			return r.GetPeriod(ctx, pid)
		}
		if !postgres.IsUniqueViolation(err, "payroll_periods_public_id_key") {
			return model.PayrollPeriod{}, fmt.Errorf("insert payroll period: %w", err)
		}
	}
	return model.PayrollPeriod{}, fmt.Errorf("no unique payroll period id after %d attempts", attempts)
}

// GetPeriod returns a period with its line count and totals.
func (r *PayrollRepository) GetPeriod(ctx context.Context, publicID string) (model.PayrollPeriod, error) {
	p, err := scanPayrollPeriod(r.pool.QueryRow(ctx, payrollPeriodSelect+` WHERE p.public_id = $1`, publicID))
	if err != nil {
		return p, fmt.Errorf("get payroll period: %w", err)
	}
	return p, nil
}

// ListPeriods returns periods whose title matches q, newest first.
func (r *PayrollRepository) ListPeriods(ctx context.Context, q string, page, perPage int) (Page[model.PayrollPeriod], error) {
	w := &Where{}
	if q != "" {
		w.Add("p.title ILIKE %s", LikePattern(q))
	}
	var out Page[model.PayrollPeriod]
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM payroll_periods p`+w.SQL(), w.Args()...).Scan(&out.Total); err != nil {
		return out, fmt.Errorf("count payroll periods: %w", err)
	}
	lim, off := w.Arg(perPage), w.Arg((page-1)*perPage)
	rows, err := r.pool.Query(ctx, payrollPeriodSelect+w.SQL()+` ORDER BY p.period_start DESC, p.created_at DESC LIMIT `+lim+` OFFSET `+off, w.Args()...)
	if err != nil {
		return out, fmt.Errorf("list payroll periods: %w", err)
	}
	out.Items, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.PayrollPeriod, error) { return scanPayrollPeriod(row) })
	if err != nil {
		return out, fmt.Errorf("scan payroll periods: %w", err)
	}
	return out, nil
}

// UpdatePeriod saves the editable fields of a period.
func (r *PayrollRepository) UpdatePeriod(ctx context.Context, p model.PayrollPeriod) (model.PayrollPeriod, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE payroll_periods SET title = $2, period_start = $3, period_end = $4, transfer_date = $5 WHERE id = $1`,
		p.ID, p.Title, p.PeriodStart, p.PeriodEnd, p.TransferDate)
	if err != nil {
		return model.PayrollPeriod{}, fmt.Errorf("update payroll period: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.PayrollPeriod{}, ErrNotFound
	}
	return r.GetPeriod(ctx, p.PublicID)
}

// DeletePeriod removes a period and its lines.
func (r *PayrollRepository) DeletePeriod(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM payroll_periods WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete payroll period: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

const payrollLineSelect = `SELECT l.id, l.period_id, l.line_no, l.crew_id, c.public_id, l.bank_account_id, l.employee_no, l.crew_name,
	l.rank_code, l.vessel_name, l.bank_name, l.account_name, l.account_number, l.amount_usd, l.amount_idr, l.transfer_date, l.remark
FROM payroll_lines l
JOIN crews c ON c.id = l.crew_id`

func scanPayrollLine(row pgx.Row) (model.PayrollLine, error) {
	var l model.PayrollLine
	err := row.Scan(&l.ID, &l.PeriodID, &l.LineNo, &l.CrewID, &l.CrewPublicID, &l.BankAccountID, &l.EmployeeNo, &l.CrewName,
		&l.RankCode, &l.VesselName, &l.BankName, &l.AccountName, &l.AccountNumber, &l.AmountUSD, &l.AmountIDR, &l.TransferDate, &l.Remark)
	return l, notFound(err)
}

var payrollLineSorts = map[string]string{
	"line_no": "l.line_no", "crew_name": "l.crew_name", "employee_no": "l.employee_no", "rank_code": "l.rank_code",
	"vessel_name": "l.vessel_name", "bank_name": "l.bank_name", "amount_usd": "l.amount_usd", "amount_idr": "l.amount_idr",
	"transfer_date": "l.transfer_date",
}

// PayrollLineSortable reports whether sort is an accepted line sort key.
func PayrollLineSortable(sort string) bool {
	_, ok := payrollLineSorts[sort]
	return ok
}

func payrollLineWhere(periodID uuid.UUID, f model.PayrollLineFilter) *Where {
	w := &Where{}
	w.Add("l.period_id = %s", periodID)
	if f.Q != "" {
		w.Add("concat_ws(' ', l.crew_name, l.employee_no, l.account_number, l.remark) ILIKE %s", LikePattern(f.Q))
	}
	if len(f.Banks) > 0 {
		w.Add("l.bank_name = ANY(%s)", f.Banks)
	}
	if len(f.Vessels) > 0 {
		w.Add("l.vessel_name = ANY(%s)", f.Vessels)
	}
	if len(f.Ranks) > 0 {
		w.Add("l.rank_code = ANY(%s)", f.Ranks)
	}
	if f.HasRemark {
		w.Add("l.remark <> ''")
	}
	if f.NoUSD {
		w.Add("l.amount_usd IS NULL")
	}
	return w
}

func (r *PayrollRepository) queryLines(ctx context.Context, w *Where, f model.PayrollLineFilter, tail string) ([]model.PayrollLine, error) {
	order := OrderWithTie(f.Sort, f.Desc, payrollLineSorts, "l.line_no", "l.line_no")
	rows, err := r.pool.Query(ctx, payrollLineSelect+w.SQL()+order+tail, w.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list payroll lines: %w", err)
	}
	lines, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.PayrollLine, error) { return scanPayrollLine(row) })
	if err != nil {
		return nil, fmt.Errorf("scan payroll lines: %w", err)
	}
	return lines, nil
}

// ListLines returns one page of the period's lines matching f.
func (r *PayrollRepository) ListLines(ctx context.Context, periodID uuid.UUID, f model.PayrollLineFilter, page, perPage int) (Page[model.PayrollLine], error) {
	w := payrollLineWhere(periodID, f)
	var out Page[model.PayrollLine]
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM payroll_lines l`+w.SQL(), w.Args()...).Scan(&out.Total); err != nil {
		return out, fmt.Errorf("count payroll lines: %w", err)
	}
	lim, off := w.Arg(perPage), w.Arg((page-1)*perPage)
	items, err := r.queryLines(ctx, w, f, ` LIMIT `+lim+` OFFSET `+off)
	out.Items = items
	return out, err
}

// AllLines returns every line of the period matching f, in the filter's order.
func (r *PayrollRepository) AllLines(ctx context.Context, periodID uuid.UUID, f model.PayrollLineFilter) ([]model.PayrollLine, error) {
	return r.queryLines(ctx, payrollLineWhere(periodID, f), f, "")
}

// LineTotals counts and sums the period's lines matching f.
func (r *PayrollRepository) LineTotals(ctx context.Context, periodID uuid.UUID, f model.PayrollLineFilter) (model.PayrollTotals, error) {
	w := payrollLineWhere(periodID, f)
	var t model.PayrollTotals
	err := r.pool.QueryRow(ctx, `SELECT count(*), coalesce(sum(l.amount_usd), 0), coalesce(sum(l.amount_idr), 0) FROM payroll_lines l`+w.SQL(),
		w.Args()...).Scan(&t.Count, &t.USD, &t.IDR)
	if err != nil {
		return t, fmt.Errorf("total payroll lines: %w", err)
	}
	return t, nil
}

// Facets lists the distinct banks, vessels and rank codes used in a period.
func (r *PayrollRepository) Facets(ctx context.Context, periodID uuid.UUID) (model.PayrollFacets, error) {
	var f model.PayrollFacets
	err := r.pool.QueryRow(ctx, `SELECT
		coalesce(array_agg(DISTINCT bank_name ORDER BY bank_name) FILTER (WHERE bank_name <> ''), '{}'),
		coalesce(array_agg(DISTINCT vessel_name ORDER BY vessel_name) FILTER (WHERE vessel_name <> ''), '{}'),
		coalesce(array_agg(DISTINCT rank_code ORDER BY rank_code) FILTER (WHERE rank_code <> ''), '{}')
		FROM payroll_lines WHERE period_id = $1`, periodID).Scan(&f.Banks, &f.Vessels, &f.Ranks)
	if err != nil {
		return f, fmt.Errorf("payroll facets: %w", err)
	}
	return f, nil
}

// GetLine returns one line.
func (r *PayrollRepository) GetLine(ctx context.Context, id uuid.UUID) (model.PayrollLine, error) {
	l, err := scanPayrollLine(r.pool.QueryRow(ctx, payrollLineSelect+` WHERE l.id = $1`, id))
	if err != nil {
		return l, fmt.Errorf("get payroll line: %w", err)
	}
	return l, nil
}

// SaveLine inserts a line at the end of its period, or updates it when it has an ID.
func (r *PayrollRepository) SaveLine(ctx context.Context, l model.PayrollLine) (model.PayrollLine, error) {
	if l.ID == uuid.Nil {
		err := r.pool.QueryRow(ctx, `INSERT INTO payroll_lines (period_id, line_no, crew_id, bank_account_id, employee_no, crew_name,
			rank_code, vessel_name, bank_name, account_name, account_number, amount_usd, amount_idr, transfer_date, remark)
			SELECT $1, coalesce(max(line_no), 0) + 1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
			FROM payroll_lines WHERE period_id = $1 RETURNING id`,
			l.PeriodID, l.CrewID, l.BankAccountID, l.EmployeeNo, l.CrewName, l.RankCode, l.VesselName, l.BankName, l.AccountName,
			l.AccountNumber, l.AmountUSD, l.AmountIDR, l.TransferDate, l.Remark).Scan(&l.ID)
		if err != nil {
			return model.PayrollLine{}, fmt.Errorf("insert payroll line: %w", err)
		}
		return r.GetLine(ctx, l.ID)
	}
	tag, err := r.pool.Exec(ctx, `UPDATE payroll_lines SET crew_id = $2, bank_account_id = $3, employee_no = $4, crew_name = $5,
		rank_code = $6, vessel_name = $7, bank_name = $8, account_name = $9, account_number = $10, amount_usd = $11, amount_idr = $12,
		transfer_date = $13, remark = $14 WHERE id = $1`,
		l.ID, l.CrewID, l.BankAccountID, l.EmployeeNo, l.CrewName, l.RankCode, l.VesselName, l.BankName, l.AccountName,
		l.AccountNumber, l.AmountUSD, l.AmountIDR, l.TransferDate, l.Remark)
	if err != nil {
		return model.PayrollLine{}, fmt.Errorf("update payroll line: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.PayrollLine{}, ErrNotFound
	}
	return r.GetLine(ctx, l.ID)
}

// DeleteLine removes one line.
func (r *PayrollRepository) DeleteLine(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM payroll_lines WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete payroll line: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// PeriodByID returns a period by internal ID.
func (r *PayrollRepository) PeriodByID(ctx context.Context, id uuid.UUID) (model.PayrollPeriod, error) {
	p, err := scanPayrollPeriod(r.pool.QueryRow(ctx, payrollPeriodSelect+` WHERE p.id = $1`, id))
	if err != nil {
		return p, fmt.Errorf("get payroll period: %w", err)
	}
	return p, nil
}

// CrewByPublicID returns the crew data a payroll line copies.
func (r *PayrollRepository) CrewByPublicID(ctx context.Context, publicID string) (model.PayrollCrew, error) {
	var c model.PayrollCrew
	err := r.pool.QueryRow(ctx, `SELECT id, public_id, full_name, employee_no FROM crews WHERE public_id = $1 AND deleted_at IS NULL`,
		publicID).Scan(&c.ID, &c.PublicID, &c.FullName, &c.EmployeeNo)
	if err != nil {
		return c, fmt.Errorf("get payroll crew: %w", notFound(err))
	}
	return c, nil
}
