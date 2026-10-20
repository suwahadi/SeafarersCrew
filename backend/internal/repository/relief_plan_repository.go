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
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
)

// ReliefPlanRepository stores relief plans and answers rotation queries.
type ReliefPlanRepository struct {
	pool *pgxpool.Pool
}

// NewReliefPlanRepository returns a ReliefPlanRepository.
func NewReliefPlanRepository(pool *pgxpool.Pool) *ReliefPlanRepository {
	return &ReliefPlanRepository{pool: pool}
}

const reliefSelect = `SELECT rp.id, rp.contract_id, rp.reliever_crew_id, rp.planned_relief_date, rp.joining_port, rp.status, rp.notes,
	rp.new_contract_id, rp.created_at, rp.updated_at, k.public_id, k.status, k.duration_months, k.rank_id, r.name, k.vessel_id, v.name,
	k.crew_id, rc.public_id, rc.full_name, rv.public_id, rv.full_name, coalesce(nk.public_id, '')
FROM relief_plans rp
JOIN contracts k ON k.id = rp.contract_id
JOIN ranks r ON r.id = k.rank_id
JOIN vessels v ON v.id = k.vessel_id
JOIN crews rc ON rc.id = k.crew_id
JOIN crews rv ON rv.id = rp.reliever_crew_id
LEFT JOIN contracts nk ON nk.id = rp.new_contract_id`

func scanReliefPlan(row pgx.Row) (model.ReliefPlan, error) {
	var p model.ReliefPlan
	var duration int16
	err := row.Scan(&p.ID, &p.ContractID, &p.RelieverCrewID, &p.PlannedReliefDate, &p.JoiningPort, &p.Status, &p.Notes,
		&p.NewContractID, &p.CreatedAt, &p.UpdatedAt, &p.ContractPublicID, &p.ContractStatus, &duration, &p.RankID, &p.RankName,
		&p.VesselID, &p.VesselName, &p.RelievedCrewID, &p.RelievedPublicID, &p.RelievedName, &p.RelieverPublicID, &p.RelieverName,
		&p.NewContractPublicID)
	p.ContractDuration = int(duration)
	return p, notFound(err)
}

// Get returns a relief plan.
func (r *ReliefPlanRepository) Get(ctx context.Context, id uuid.UUID) (model.ReliefPlan, error) {
	p, err := scanReliefPlan(r.pool.QueryRow(ctx, reliefSelect+` WHERE rp.id = $1`, id))
	if err != nil {
		return p, fmt.Errorf("get relief plan: %w", err)
	}
	return p, nil
}

// List returns one page of relief plans ordered by planned relief date.
func (r *ReliefPlanRepository) List(ctx context.Context, f model.ReliefPlanFilter, limit, offset int) (Page[model.ReliefPlan], error) {
	w := &Where{}
	if len(f.Statuses) > 0 {
		w.Add("rp.status = ANY(%s)", f.Statuses)
	}
	if f.ContractPublicID != "" {
		w.Add("k.public_id = %s", strings.ToUpper(f.ContractPublicID))
	}
	if f.RelieverPublicID != "" {
		w.Add("rv.public_id = %s", strings.ToUpper(f.RelieverPublicID))
	}
	if len(f.VesselIDs) > 0 {
		w.Add("k.vessel_id = ANY(%s)", f.VesselIDs)
	}
	if f.DateFrom != nil {
		w.Add("rp.planned_relief_date >= %s", *f.DateFrom)
	}
	if f.DateTo != nil {
		w.Add("rp.planned_relief_date <= %s", *f.DateTo)
	}
	var page Page[model.ReliefPlan]
	count := `SELECT count(*) FROM relief_plans rp JOIN contracts k ON k.id = rp.contract_id JOIN crews rv ON rv.id = rp.reliever_crew_id`
	if err := r.pool.QueryRow(ctx, count+w.SQL(), w.Args()...).Scan(&page.Total); err != nil {
		return page, fmt.Errorf("count relief plans: %w", err)
	}
	lim, off := w.Arg(limit), w.Arg(offset)
	rows, err := r.pool.Query(ctx, reliefSelect+w.SQL()+` ORDER BY rp.planned_relief_date, rp.id LIMIT `+lim+` OFFSET `+off, w.Args()...)
	if err != nil {
		return page, fmt.Errorf("list relief plans: %w", err)
	}
	page.Items, err = collect(rows, scanReliefPlan, "relief plans")
	return page, err
}

// Create inserts a relief plan.
func (r *ReliefPlanRepository) Create(ctx context.Context, p model.ReliefPlan) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `INSERT INTO relief_plans (contract_id, reliever_crew_id, planned_relief_date, joining_port, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, p.ContractID, p.RelieverCrewID, p.PlannedReliefDate, p.JoiningPort, p.Status, p.Notes).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert relief plan: %w", err)
	}
	return id, nil
}

// Update changes an open (planned or confirmed) relief plan.
func (r *ReliefPlanRepository) Update(ctx context.Context, p model.ReliefPlan) error {
	tag, err := r.pool.Exec(ctx, `UPDATE relief_plans SET reliever_crew_id = $2, planned_relief_date = $3, joining_port = $4, status = $5, notes = $6
		WHERE id = $1 AND status IN ('planned', 'confirmed')`, p.ID, p.RelieverCrewID, p.PlannedReliefDate, p.JoiningPort, p.Status, p.Notes)
	if err != nil {
		return fmt.Errorf("update relief plan: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrStateChanged
	}
	return nil
}

// Delete removes a relief plan that has not been completed.
func (r *ReliefPlanRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM relief_plans WHERE id = $1 AND status <> 'completed'`, id)
	if err != nil {
		return fmt.Errorf("delete relief plan: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrStateChanged
	}
	return nil
}

// OpenWindows returns the planned and confirmed commitments of a reliever, excluding one plan.
func (r *ReliefPlanRepository) OpenWindows(ctx context.Context, relieverID, excludePlanID uuid.UUID) ([]model.ReliefWindow, error) {
	rows, err := r.pool.Query(ctx, `SELECT rp.id, rp.planned_relief_date, k.duration_months FROM relief_plans rp
		JOIN contracts k ON k.id = rp.contract_id
		WHERE rp.reliever_crew_id = $1 AND rp.id <> $2 AND rp.status IN ('planned', 'confirmed')`, relieverID, excludePlanID)
	if err != nil {
		return nil, fmt.Errorf("list reliever windows: %w", err)
	}
	return collect(rows, func(row pgx.Row) (model.ReliefWindow, error) {
		var w model.ReliefWindow
		var months int16
		err := row.Scan(&w.PlanID, &w.ReliefDate, &months)
		w.DurationMonths = int(months)
		return w, err
	}, "reliever windows")
}

// NextReliefDate returns the earliest open relief date planned for a contract.
func (r *ReliefPlanRepository) NextReliefDate(ctx context.Context, contractID uuid.UUID) (*time.Time, error) {
	var d *time.Time
	err := r.pool.QueryRow(ctx, `SELECT min(planned_relief_date) FROM relief_plans WHERE contract_id = $1 AND status IN ('planned', 'confirmed')`, contractID).Scan(&d)
	if err != nil {
		return nil, fmt.Errorf("next relief date: %w", err)
	}
	return d, nil
}

// Suggestions lists Available crew of rankID, compliant for that rank on date and free of overlapping reliefs until end,
// sorted by merged sea service in the rank.
func (r *ReliefPlanRepository) Suggestions(ctx context.Context, rankID, excludeCrewID uuid.UUID, date, end time.Time, limit int) ([]model.ReliefCandidate, error) {
	rows, err := r.pool.Query(ctx, `SELECT c.id, c.public_id, c.full_name, c.nationality, c.date_of_birth, c.mobile_phone,
		`+seaDaysExpr("$3::date", " AND e.rank_id = $1")+` AS sea_days
		FROM crews c
		JOIN crew_statuses s ON s.id = c.crew_status_id
		LEFT JOIN contracts ac ON ac.crew_id = c.id AND ac.status = 'active' AND ac.deleted_at IS NULL
		LEFT JOIN ranks er ON er.id = coalesce(ac.rank_id, c.applied_rank_id)
		WHERE c.deleted_at IS NULL AND s.code = 'available' AND ac.id IS NULL AND c.applied_rank_id = $1 AND c.id <> $2
		AND `+compliantExpr("$3::date")+`
		AND NOT EXISTS (
			SELECT 1 FROM relief_plans rp JOIN contracts rk ON rk.id = rp.contract_id
			WHERE rp.reliever_crew_id = c.id AND rp.status IN ('planned', 'confirmed')
			AND rp.planned_relief_date <= $4::date
			AND (rp.planned_relief_date + make_interval(months => CASE WHEN rk.duration_months > 0 THEN rk.duration_months ELSE 6 END))::date >= $3::date)
		ORDER BY sea_days DESC, c.full_name, c.id
		LIMIT $5`, rankID, excludeCrewID, date, end, limit)
	if err != nil {
		return nil, fmt.Errorf("relief suggestions: %w", err)
	}
	return collect(rows, func(row pgx.Row) (model.ReliefCandidate, error) {
		var c model.ReliefCandidate
		err := row.Scan(&c.CrewID, &c.PublicID, &c.FullName, &c.Nationality, &c.DateOfBirth, &c.MobilePhone, &c.SeaServiceDays)
		return c, err
	}, "relief suggestions")
}

// TimelineVessels returns the active vessels shown as lanes.
func (r *ReliefPlanRepository) TimelineVessels(ctx context.Context, vesselIDs []uuid.UUID) ([]model.TimelineVessel, error) {
	w := &Where{}
	w.Add("v.deleted_at IS NULL AND v.is_active AND v.status = 'active'")
	if len(vesselIDs) > 0 {
		w.Add("v.id = ANY(%s)", vesselIDs)
	}
	rows, err := r.pool.Query(ctx, `SELECT v.id, v.name, coalesce(vt.name, '') FROM vessels v LEFT JOIN vessel_types vt ON vt.id = v.vessel_type_id`+
		w.SQL()+` ORDER BY v.sort_order, v.name`, w.Args()...)
	if err != nil {
		return nil, fmt.Errorf("timeline vessels: %w", err)
	}
	return collect(rows, func(row pgx.Row) (model.TimelineVessel, error) {
		var v model.TimelineVessel
		err := row.Scan(&v.ID, &v.Name, &v.Type)
		return v, err
	}, "timeline vessels")
}

// TimelineBars returns the Active contracts overlapping [from, to].
func (r *ReliefPlanRepository) TimelineBars(ctx context.Context, from, to time.Time, vesselIDs []uuid.UUID) ([]model.TimelineBar, error) {
	w := &Where{}
	w.Add("k.status = 'active' AND k.deleted_at IS NULL")
	w.Add("k.sign_on_date <= %s AND k.planned_sign_off_date >= %s", to, from)
	if len(vesselIDs) > 0 {
		w.Add("k.vessel_id = ANY(%s)", vesselIDs)
	}
	rows, err := r.pool.Query(ctx, `SELECT k.id, k.public_id, k.vessel_id, v.name, c.public_id, c.full_name, r.name, k.sign_on_date, k.planned_sign_off_date
		FROM contracts k JOIN crews c ON c.id = k.crew_id JOIN ranks r ON r.id = k.rank_id JOIN vessels v ON v.id = k.vessel_id`+w.SQL()+` ORDER BY r.sort_order, k.sign_on_date`, w.Args()...)
	if err != nil {
		return nil, fmt.Errorf("timeline bars: %w", err)
	}
	return collect(rows, func(row pgx.Row) (model.TimelineBar, error) {
		var b model.TimelineBar
		err := row.Scan(&b.ContractID, &b.ContractPublicID, &b.VesselID, &b.VesselName, &b.CrewPublicID, &b.CrewName, &b.RankName, &b.SignOnDate, &b.PlannedSignOffDate)
		return b, err
	}, "timeline bars")
}

// OpenPlansForContracts returns planned and confirmed relief plans of the given contracts.
func (r *ReliefPlanRepository) OpenPlansForContracts(ctx context.Context, contractIDs []uuid.UUID) ([]model.ReliefPlan, error) {
	if len(contractIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, reliefSelect+` WHERE rp.contract_id = ANY($1) AND rp.status IN ('planned', 'confirmed') ORDER BY rp.planned_relief_date`, contractIDs)
	if err != nil {
		return nil, fmt.Errorf("open relief plans: %w", err)
	}
	return collect(rows, scanReliefPlan, "open relief plans")
}

// ErrPlanCancelled is returned when completing a cancelled relief plan.
var ErrPlanCancelled = errors.New("relief plan is cancelled")

// Complete marks a plan completed and creates the reliever's Draft contract from the relieved one; a repeated call
// returns the contract created the first time.
func (r *ReliefPlanRepository) Complete(ctx context.Context, planID uuid.UUID, newID func() (string, error), attempts int) (uuid.UUID, error) {
	var contractID uuid.UUID
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		var status string
		var existing *uuid.UUID
		var relieverID, sourceID uuid.UUID
		var reliefDate time.Time
		var port string
		err := tx.QueryRow(ctx, `SELECT status, new_contract_id, reliever_crew_id, contract_id, planned_relief_date, joining_port
			FROM relief_plans WHERE id = $1 FOR UPDATE`, planID).Scan(&status, &existing, &relieverID, &sourceID, &reliefDate, &port)
		if err != nil {
			return fmt.Errorf("lock relief plan: %w", notFound(err))
		}
		if existing != nil {
			contractID = *existing
			return nil
		}
		if status == model.ReliefCancelled || status == model.ReliefCompleted {
			return ErrPlanCancelled
		}
		src, err := loadContractForCopy(ctx, tx, sourceID)
		if err != nil {
			return err
		}
		next := src
		next.CrewID = relieverID
		next.SignOnDate = reliefDate
		next.PlannedSignOffDate = datetime.AddMonths(reliefDate, src.DurationMonths)
		next.SignOnPort = port
		next.SignOffPort = ""
		next.Notes = ""
		contractID, err = insertWithPublicID(ctx, tx, next, newID, attempts)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE relief_plans SET status = 'completed', new_contract_id = $2 WHERE id = $1`, planID, contractID)
		if err != nil {
			return fmt.Errorf("complete relief plan: %w", err)
		}
		return nil
	})
	return contractID, err
}

func loadContractForCopy(ctx context.Context, tx pgx.Tx, id uuid.UUID) (model.Contract, error) {
	var k model.Contract
	var duration int16
	var hours, probation *int16
	err := tx.QueryRow(ctx, `SELECT vessel_id, rank_id, principal_id, employer_company_id, duration_months, point_of_hire,
		currency_code, basic_salary, working_hours_week, leave_days_month, probation_months
		FROM contracts WHERE id = $1`, id).Scan(&k.VesselID, &k.RankID, &k.PrincipalID, &k.EmployerCompanyID, &duration, &k.PointOfHire,
		&k.CurrencyCode, &k.BasicSalary, &hours, &k.LeaveDaysMonth, &probation)
	if err != nil {
		return k, fmt.Errorf("load relieved contract: %w", notFound(err))
	}
	k.DurationMonths = int(duration)
	k.WorkingHoursWeek = intPtr(hours)
	k.ProbationMonths = intPtr(probation)
	rows, err := tx.Query(ctx, `SELECT allowance_type_id, basis, value, monthly_amount FROM contract_allowances WHERE contract_id = $1 ORDER BY sort_order`, id)
	if err != nil {
		return k, fmt.Errorf("load relieved allowances: %w", err)
	}
	k.Allowances, err = collect(rows, func(row pgx.Row) (model.ContractAllowance, error) {
		var a model.ContractAllowance
		err := row.Scan(&a.AllowanceTypeID, &a.Basis, &a.Value, &a.MonthlyAmount)
		return a, err
	}, "relieved allowances")
	return k, err
}

func insertWithPublicID(ctx context.Context, tx pgx.Tx, k model.Contract, newID func() (string, error), attempts int) (uuid.UUID, error) {
	for range attempts {
		pid, err := newID()
		if err != nil {
			return uuid.Nil, err
		}
		sp, err := tx.Begin(ctx)
		if err != nil {
			return uuid.Nil, fmt.Errorf("savepoint: %w", err)
		}
		id, err := insertContract(ctx, sp, k, pid)
		if err == nil {
			if err := sp.Commit(ctx); err != nil {
				return uuid.Nil, fmt.Errorf("release savepoint: %w", err)
			}
			return id, nil
		}
		if rbErr := sp.Rollback(ctx); rbErr != nil {
			return uuid.Nil, fmt.Errorf("rollback savepoint: %w", rbErr)
		}
		if !postgres.IsUniqueViolation(err, "contracts_public_id_key") {
			return uuid.Nil, err
		}
	}
	return uuid.Nil, fmt.Errorf("no unique contract id after %d attempts", attempts)
}
