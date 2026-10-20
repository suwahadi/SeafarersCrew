package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
)

// ContractRepository stores contracts and their allowances.
type ContractRepository struct {
	pool *pgxpool.Pool
}

// NewContractRepository returns a ContractRepository.
func NewContractRepository(pool *pgxpool.Pool) *ContractRepository {
	return &ContractRepository{pool: pool}
}

// ErrStateChanged is returned when a contract is no longer in the state an operation expects.
var ErrStateChanged = errors.New("contract state changed")

const contractFrom = `
FROM contracts k
JOIN crews c ON c.id = k.crew_id
JOIN crew_statuses s ON s.id = c.crew_status_id
JOIN vessels v ON v.id = k.vessel_id
LEFT JOIN vessel_types vt ON vt.id = v.vessel_type_id
JOIN ranks r ON r.id = k.rank_id
JOIN companies p ON p.id = k.principal_id
JOIN companies e ON e.id = k.employer_company_id
JOIN currencies cur ON cur.code = k.currency_code`

const contractColumns = `k.id, k.public_id, k.crew_id, k.vessel_id, k.rank_id, k.principal_id, k.employer_company_id,
	k.sign_on_date, k.duration_months, k.planned_sign_off_date, k.actual_sign_off_date, k.point_of_hire, k.sign_on_port,
	k.sign_off_port, k.currency_code, k.basic_salary, k.working_hours_week, k.leave_days_month, k.probation_months,
	k.status, k.termination_reason, k.notes, k.created_at, k.updated_at,
	c.public_id, c.full_name, coalesce(s.code, ''), v.name, v.vessel_type_id, coalesce(vt.name, ''), v.grt, v.hp_kw,
	r.name, p.name, e.name, cur.decimals,
	k.basic_salary + coalesce((SELECT sum(a.monthly_amount) FROM contract_allowances a WHERE a.contract_id = k.id), 0)`

func scanContract(row pgx.Row) (model.Contract, error) {
	var k model.Contract
	var duration int16
	var hours, probation *int16
	var decimals int16
	err := row.Scan(&k.ID, &k.PublicID, &k.CrewID, &k.VesselID, &k.RankID, &k.PrincipalID, &k.EmployerCompanyID,
		&k.SignOnDate, &duration, &k.PlannedSignOffDate, &k.ActualSignOffDate, &k.PointOfHire, &k.SignOnPort,
		&k.SignOffPort, &k.CurrencyCode, &k.BasicSalary, &hours, &k.LeaveDaysMonth, &probation,
		&k.Status, &k.TerminationReason, &k.Notes, &k.CreatedAt, &k.UpdatedAt,
		&k.CrewPublicID, &k.CrewName, &k.CrewStatusCode, &k.VesselName, &k.VesselTypeID, &k.VesselTypeName, &k.VesselGRT, &k.VesselHPKW,
		&k.RankName, &k.PrincipalName, &k.EmployerName, &decimals, &k.TotalMonthlyPay)
	if err != nil {
		return k, notFound(err)
	}
	k.DurationMonths = int(duration)
	k.WorkingHoursWeek = intPtr(hours)
	k.ProbationMonths = intPtr(probation)
	k.CurrencyDecimals = int(decimals)
	return k, nil
}

func intPtr(v *int16) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}

// Create inserts a contract with its allowances.
func (r *ContractRepository) Create(ctx context.Context, k model.Contract, publicID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		id, err = insertContract(ctx, tx, k, publicID)
		return err
	})
	return id, err
}

func insertContract(ctx context.Context, tx pgx.Tx, k model.Contract, publicID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `INSERT INTO contracts (public_id, crew_id, vessel_id, rank_id, principal_id, employer_company_id,
		sign_on_date, duration_months, planned_sign_off_date, point_of_hire, sign_on_port, sign_off_port, currency_code,
		basic_salary, working_hours_week, leave_days_month, probation_months, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, 'draft', $18) RETURNING id`,
		publicID, k.CrewID, k.VesselID, k.RankID, k.PrincipalID, k.EmployerCompanyID, k.SignOnDate, k.DurationMonths,
		k.PlannedSignOffDate, k.PointOfHire, k.SignOnPort, k.SignOffPort, k.CurrencyCode, k.BasicSalary, k.WorkingHoursWeek,
		k.LeaveDaysMonth, k.ProbationMonths, k.Notes).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert contract: %w", err)
	}
	if err := insertAllowances(ctx, tx, id, k.Allowances); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func insertAllowances(ctx context.Context, tx pgx.Tx, contractID uuid.UUID, rows []model.ContractAllowance) error {
	for i, a := range rows {
		if _, err := tx.Exec(ctx, `INSERT INTO contract_allowances (contract_id, allowance_type_id, basis, value, monthly_amount, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6)`, contractID, a.AllowanceTypeID, a.Basis, a.Value, a.MonthlyAmount, (i+1)*10); err != nil {
			return fmt.Errorf("insert contract allowance: %w", err)
		}
	}
	return nil
}

// Update replaces the editable fields and allowances of a contract that is still in status.
func (r *ContractRepository) Update(ctx context.Context, k model.Contract, status string) error {
	return InTx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE contracts SET crew_id = $3, vessel_id = $4, rank_id = $5, principal_id = $6,
			employer_company_id = $7, sign_on_date = $8, duration_months = $9, planned_sign_off_date = $10, point_of_hire = $11,
			sign_on_port = $12, sign_off_port = $13, currency_code = $14, basic_salary = $15, working_hours_week = $16,
			leave_days_month = $17, probation_months = $18, notes = $19
			WHERE id = $1 AND status = $2 AND deleted_at IS NULL`,
			k.ID, status, k.CrewID, k.VesselID, k.RankID, k.PrincipalID, k.EmployerCompanyID, k.SignOnDate, k.DurationMonths,
			k.PlannedSignOffDate, k.PointOfHire, k.SignOnPort, k.SignOffPort, k.CurrencyCode, k.BasicSalary, k.WorkingHoursWeek,
			k.LeaveDaysMonth, k.ProbationMonths, k.Notes)
		if err != nil {
			return fmt.Errorf("update contract: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrStateChanged
		}
		if _, err := tx.Exec(ctx, `DELETE FROM contract_allowances WHERE contract_id = $1`, k.ID); err != nil {
			return fmt.Errorf("clear contract allowances: %w", err)
		}
		return insertAllowances(ctx, tx, k.ID, k.Allowances)
	})
}

// GetByPublicID returns a contract with its allowances.
func (r *ContractRepository) GetByPublicID(ctx context.Context, publicID string) (model.Contract, error) {
	return r.get(ctx, `k.public_id = $1`, publicID)
}

// GetByID returns a contract with its allowances.
func (r *ContractRepository) GetByID(ctx context.Context, id uuid.UUID) (model.Contract, error) {
	return r.get(ctx, `k.id = $1`, id)
}

// ActiveForCrew returns the Active contract of a crew member.
func (r *ContractRepository) ActiveForCrew(ctx context.Context, crewID uuid.UUID) (model.Contract, error) {
	return r.get(ctx, `k.crew_id = $1 AND k.status = 'active'`, crewID)
}

func (r *ContractRepository) get(ctx context.Context, cond string, arg any) (model.Contract, error) {
	k, err := scanContract(r.pool.QueryRow(ctx, `SELECT `+contractColumns+contractFrom+` WHERE `+cond+` AND k.deleted_at IS NULL`, arg))
	if err != nil {
		return k, fmt.Errorf("get contract: %w", err)
	}
	k.Allowances, err = r.allowances(ctx, k.ID)
	return k, err
}

func (r *ContractRepository) allowances(ctx context.Context, contractID uuid.UUID) ([]model.ContractAllowance, error) {
	rows, err := r.pool.Query(ctx, `SELECT a.id, a.allowance_type_id, t.name, a.basis, a.value, a.monthly_amount, a.sort_order
		FROM contract_allowances a JOIN allowance_types t ON t.id = a.allowance_type_id
		WHERE a.contract_id = $1 ORDER BY a.sort_order, a.created_at`, contractID)
	if err != nil {
		return nil, fmt.Errorf("list contract allowances: %w", err)
	}
	return collect(rows, func(row pgx.Row) (model.ContractAllowance, error) {
		var a model.ContractAllowance
		err := row.Scan(&a.ID, &a.AllowanceTypeID, &a.AllowanceType, &a.Basis, &a.Value, &a.MonthlyAmount, &a.SortOrder)
		return a, err
	}, "contract allowances")
}

// List returns one page of contracts matching f.
func (r *ContractRepository) List(ctx context.Context, f model.ContractFilter, limit, offset int) (Page[model.Contract], error) {
	w := contractWhere(f)
	var page Page[model.Contract]
	if err := r.pool.QueryRow(ctx, `SELECT count(*)`+contractFrom+w.SQL(), w.Args()...).Scan(&page.Total); err != nil {
		return page, fmt.Errorf("count contracts: %w", err)
	}
	cols := map[string]string{
		"id": "k.public_id", "crew_name": "c.full_name", "vessel": "v.name", "rank": "r.sort_order", "principal": "p.name",
		"sign_on_date": "k.sign_on_date", "planned_sign_off_date": "k.planned_sign_off_date", "status": "k.status",
		"basic_salary": "k.basic_salary", "created_at": "k.created_at",
	}
	lim, off := w.Arg(limit), w.Arg(offset)
	rows, err := r.pool.Query(ctx, `SELECT `+contractColumns+contractFrom+w.SQL()+
		OrderWithTie(f.Sort, f.Desc, cols, "k.created_at DESC, k.id DESC", "k.id")+` LIMIT `+lim+` OFFSET `+off, w.Args()...)
	if err != nil {
		return page, fmt.Errorf("list contracts: %w", err)
	}
	page.Items, err = collect(rows, scanContract, "contracts")
	return page, err
}

func contractWhere(f model.ContractFilter) *Where {
	w := &Where{}
	w.Add("k.deleted_at IS NULL")
	if f.Q != "" {
		like := LikePattern(f.Q)
		w.Add("(k.public_id = %s OR c.public_id = %s OR c.full_name ILIKE %s OR v.name ILIKE %s)", strings.ToUpper(f.Q), strings.ToUpper(f.Q), like, like)
	}
	if len(f.Statuses) > 0 {
		w.Add("k.status = ANY(%s)", f.Statuses)
	}
	if len(f.VesselIDs) > 0 {
		w.Add("k.vessel_id = ANY(%s)", f.VesselIDs)
	}
	if len(f.PrincipalIDs) > 0 {
		w.Add("k.principal_id = ANY(%s)", f.PrincipalIDs)
	}
	if len(f.RankIDs) > 0 {
		w.Add("k.rank_id = ANY(%s)", f.RankIDs)
	}
	if f.CrewPublicID != "" {
		w.Add("c.public_id = %s", strings.ToUpper(f.CrewPublicID))
	}
	if f.SignOnFrom != nil {
		w.Add("k.sign_on_date >= %s", *f.SignOnFrom)
	}
	if f.SignOnTo != nil {
		w.Add("k.sign_on_date <= %s", *f.SignOnTo)
	}
	if f.SignOffBefore != nil {
		w.Add("k.planned_sign_off_date <= %s", *f.SignOffBefore)
	}
	return w
}

// SoftDelete marks a draft contract deleted.
func (r *ContractRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE contracts SET deleted_at = now() WHERE id = $1 AND status = 'draft' AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("delete contract: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrStateChanged
	}
	return nil
}

// Activate makes a draft contract Active and puts the crew member On Board in one transaction.
func (r *ContractRepository) Activate(ctx context.Context, id, crewID uuid.UUID) error {
	return InTx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE contracts SET status = 'active' WHERE id = $1 AND status = 'draft' AND deleted_at IS NULL`, id)
		if err != nil {
			if postgres.IsUniqueViolation(err, "contracts_one_active_per_crew") {
				return ErrCrewHasActiveContract
			}
			return fmt.Errorf("activate contract: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrStateChanged
		}
		return setCrewStatus(ctx, tx, crewID, "on_board")
	})
}

// ErrCrewHasActiveContract is returned when the one-active-contract index rejects an activation.
var ErrCrewHasActiveContract = errors.New("crew already holds an active contract")

// Finish completes or terminates an Active contract, records exactly one Experience and signs the crew member off.
func (r *ContractRepository) Finish(ctx context.Context, id, crewID uuid.UUID, status string, actual time.Time, reason string, x model.Experience) error {
	return InTx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE contracts SET status = $2, actual_sign_off_date = $3, termination_reason = $4
			WHERE id = $1 AND status = 'active' AND deleted_at IS NULL`, id, status, actual, reason)
		if err != nil {
			return fmt.Errorf("finish contract: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrStateChanged
		}
		if err := RecordContractExperience(ctx, tx, x); err != nil {
			return err
		}
		return setCrewStatus(ctx, tx, crewID, "sign_off")
	})
}

// RecordContractExperience inserts the Experience of a contract once; repeated calls leave the first row untouched.
func RecordContractExperience(ctx context.Context, db DBTX, x model.Experience) error {
	_, err := db.Exec(ctx, `INSERT INTO experiences (crew_id, vessel_name, vessel_id, rank_id, vessel_type_id, grt_hp, company_id,
		company_name, principal_id, principal_name, sign_on_date, sign_off_date, status, remarks, source, contract_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'sign_off', $13, 'contract', $14)
		ON CONFLICT (contract_id) DO NOTHING`,
		x.CrewID, x.VesselName, x.VesselID, x.RankID, x.VesselTypeID, x.GRTHP, x.CompanyID, x.CompanyName, x.PrincipalID,
		x.PrincipalName, x.SignOnDate, x.SignOffDate, x.Remarks, x.ContractID)
	if err != nil {
		return fmt.Errorf("record contract experience: %w", err)
	}
	return nil
}

func setCrewStatus(ctx context.Context, tx pgx.Tx, crewID uuid.UUID, code string) error {
	tag, err := tx.Exec(ctx, `UPDATE crews SET crew_status_id = s.id, status_reason = ''
		FROM crew_statuses s WHERE crews.id = $1 AND s.code = $2`, crewID, code)
	if err != nil {
		return fmt.Errorf("set crew status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("set crew status %q: %w", code, ErrNotFound)
	}
	return nil
}

// ContractLookupRepository answers existence questions used by contract rules.
type ContractLookupRepository struct {
	pool *pgxpool.Pool
}

// NewContractLookupRepository returns a ContractLookupRepository.
func NewContractLookupRepository(pool *pgxpool.Pool) *ContractLookupRepository {
	return &ContractLookupRepository{pool: pool}
}

// CompanyType returns the type of an active company.
func (r *ContractLookupRepository) CompanyType(ctx context.Context, id uuid.UUID) (string, error) {
	var t string
	err := r.pool.QueryRow(ctx, `SELECT type FROM companies WHERE id = $1 AND deleted_at IS NULL AND is_active`, id).Scan(&t)
	if err != nil {
		return "", fmt.Errorf("company type: %w", notFound(err))
	}
	return t, nil
}

// VesselActive reports whether an active, non-deleted vessel exists.
func (r *ContractLookupRepository) VesselActive(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.exists(ctx, `SELECT EXISTS (SELECT 1 FROM vessels WHERE id = $1 AND deleted_at IS NULL AND is_active)`, id)
}

// RankActive reports whether an active, non-deleted rank exists.
func (r *ContractLookupRepository) RankActive(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.exists(ctx, `SELECT EXISTS (SELECT 1 FROM ranks WHERE id = $1 AND deleted_at IS NULL AND is_active)`, id)
}

// CurrencyDecimals returns the decimal places of an active currency.
func (r *ContractLookupRepository) CurrencyDecimals(ctx context.Context, code string) (int, error) {
	var d int16
	err := r.pool.QueryRow(ctx, `SELECT decimals FROM currencies WHERE code = $1 AND deleted_at IS NULL AND is_active`, code).Scan(&d)
	if err != nil {
		return 0, fmt.Errorf("currency: %w", notFound(err))
	}
	return int(d), nil
}

// AllowanceType returns the name and basis of an active allowance type.
func (r *ContractLookupRepository) AllowanceType(ctx context.Context, id uuid.UUID) (string, string, error) {
	var name, basis string
	err := r.pool.QueryRow(ctx, `SELECT name, basis FROM allowance_types WHERE id = $1 AND deleted_at IS NULL AND is_active`, id).Scan(&name, &basis)
	if err != nil {
		return "", "", fmt.Errorf("allowance type: %w", notFound(err))
	}
	return name, basis, nil
}

func (r *ContractLookupRepository) exists(ctx context.Context, sql string, id uuid.UUID) (bool, error) {
	var ok bool
	if err := r.pool.QueryRow(ctx, sql, id).Scan(&ok); err != nil {
		return false, fmt.Errorf("lookup: %w", err)
	}
	return ok, nil
}
