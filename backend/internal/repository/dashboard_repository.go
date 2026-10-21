package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// DashboardRepository runs the aggregate queries of the dashboard.
type DashboardRepository struct {
	pool *pgxpool.Pool
}

// NewDashboardRepository returns a DashboardRepository.
func NewDashboardRepository(pool *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{pool: pool}
}

const latestExpiryPerType = `
SELECT 'document' AS kind, td.crew_id, dt.name AS type_name, td.number, td.expired_date
FROM (SELECT DISTINCT ON (crew_id, document_type_id) crew_id, document_type_id, number, expired_date
      FROM travel_documents WHERE expired_date IS NOT NULL
      ORDER BY crew_id, document_type_id, expired_date DESC) td
JOIN document_types dt ON dt.id = td.document_type_id AND dt.has_expiry AND dt.deleted_at IS NULL
UNION ALL
SELECT 'certificate', ce.crew_id, ct.name, ce.number, ce.expired_date
FROM (SELECT DISTINCT ON (crew_id, certificate_type_id) crew_id, certificate_type_id, number, expired_date
      FROM certificates WHERE expired_date IS NOT NULL
      ORDER BY crew_id, certificate_type_id, expired_date DESC) ce
JOIN certificate_types ct ON ct.id = ce.certificate_type_id AND ct.has_expiry AND ct.deleted_at IS NULL`

// Summary computes the KPI cards for today with the expiry thresholds.
func (r *DashboardRepository) Summary(ctx context.Context, today time.Time, t [3]int) (model.DashboardSummary, error) {
	s := model.DashboardSummary{Thresholds: t}
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		s.ByStatus, err = r.crewByStatus(gctx)
		for _, b := range s.ByStatus {
			s.TotalCrew += b.Count
		}
		return err
	})
	g.Go(func() error {
		return r.pool.QueryRow(gctx, `SELECT
			(SELECT count(*) FROM contracts WHERE status = 'active' AND deleted_at IS NULL),
			(SELECT count(*) FROM contracts c JOIN crews cr ON cr.id = c.crew_id AND cr.deleted_at IS NULL
			 WHERE c.status = 'active' AND c.deleted_at IS NULL AND c.planned_sign_off_date <= $1::date + 30),
			(SELECT count(*) FROM applicants WHERE stage = 'new' AND deleted_at IS NULL),
			(SELECT count(*) FROM contact_messages WHERE NOT is_read AND NOT is_archived)`, today).
			Scan(&s.ActiveContracts, &s.DueForRelief, &s.NewApplicants, &s.UnreadMessages)
	})
	g.Go(func() error {
		rows, err := r.pool.Query(gctx, `SELECT x.kind,
			count(*) FILTER (WHERE x.expired_date < $1::date),
			count(*) FILTER (WHERE x.expired_date >= $1::date AND x.expired_date <= $1::date + $2::int),
			count(*) FILTER (WHERE x.expired_date > $1::date + $2::int AND x.expired_date <= $1::date + $3::int),
			count(*) FILTER (WHERE x.expired_date > $1::date + $3::int AND x.expired_date <= $1::date + $4::int)
			FROM (`+latestExpiryPerType+`) x JOIN crews c ON c.id = x.crew_id AND c.deleted_at IS NULL
			GROUP BY x.kind`, today, t[0], t[1], t[2])
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var kind string
			var b model.ExpiryBuckets
			if err := rows.Scan(&kind, &b.Expired, &b.Within1, &b.Within2, &b.Within3); err != nil {
				return err
			}
			if kind == "document" {
				s.Documents = b
			} else {
				s.Certificates = b
			}
		}
		return rows.Err()
	})
	g.Go(func() error {
		var status *string
		err := r.pool.QueryRow(gctx, `SELECT status, finished_at FROM backups ORDER BY started_at DESC LIMIT 1`).Scan(&status, &s.LastBackupFinished)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if status != nil {
			s.LastBackupStatus = *status
		}
		return err
	})
	if err := g.Wait(); err != nil {
		return s, fmt.Errorf("dashboard summary: %w", err)
	}
	return s, nil
}

func (r *DashboardRepository) crewByStatus(ctx context.Context) ([]model.CountByKey, error) {
	rows, err := r.pool.Query(ctx, `SELECT coalesce(s.code, s.id::text), s.name, s.color, count(c.id)
		FROM crew_statuses s LEFT JOIN crews c ON c.crew_status_id = s.id AND c.deleted_at IS NULL
		WHERE s.deleted_at IS NULL GROUP BY s.id ORDER BY s.sort_order, s.name`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.CountByKey, error) {
		var c model.CountByKey
		err := row.Scan(&c.Key, &c.Label, &c.Color, &c.Count)
		return c, err
	})
}

// Charts computes the dashboard chart series; movements cover the 12 months ending in today's month.
func (r *DashboardRepository) Charts(ctx context.Context, today time.Time) (model.DashboardCharts, error) {
	var ch model.DashboardCharts
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		ch.ByStatus, err = r.crewByStatus(gctx)
		return err
	})
	g.Go(func() error {
		rows, err := r.pool.Query(gctx, `SELECT r.id::text, r.name, r.department, count(c.id)
			FROM ranks r
			JOIN crews c ON c.deleted_at IS NULL AND r.id = coalesce(
				(SELECT ac.rank_id FROM contracts ac WHERE ac.crew_id = c.id AND ac.status = 'active' AND ac.deleted_at IS NULL), c.applied_rank_id)
			WHERE r.deleted_at IS NULL
			GROUP BY r.id ORDER BY CASE r.department WHEN 'deck' THEN 1 WHEN 'engine' THEN 2 ELSE 3 END, r.sort_order, r.name`)
		if err != nil {
			return err
		}
		ch.ByRank, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.CountByKey, error) {
			var c model.CountByKey
			err := row.Scan(&c.Key, &c.Label, &c.Group, &c.Count)
			return c, err
		})
		return err
	})
	g.Go(func() error {
		rows, err := r.pool.Query(gctx, `SELECT p.id::text, p.name, count(*)
			FROM contracts c JOIN companies p ON p.id = c.principal_id
			JOIN crews cr ON cr.id = c.crew_id AND cr.deleted_at IS NULL
			WHERE c.status = 'active' AND c.deleted_at IS NULL
			GROUP BY p.id ORDER BY count(*) DESC, p.name`)
		if err != nil {
			return err
		}
		ch.ByPrincipal, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.CountByKey, error) {
			var c model.CountByKey
			err := row.Scan(&c.Key, &c.Label, &c.Count)
			return c, err
		})
		return err
	})
	g.Go(func() error {
		rows, err := r.pool.Query(gctx, `WITH months AS (
				SELECT generate_series(date_trunc('month', $1::date) - interval '11 months', date_trunc('month', $1::date), interval '1 month')::date AS m)
			SELECT to_char(m.m, 'YYYY-MM'),
				(SELECT count(*) FROM contracts c WHERE c.deleted_at IS NULL AND c.status IN ('active', 'completed', 'terminated')
				 AND date_trunc('month', c.sign_on_date) = m.m),
				(SELECT count(*) FROM contracts c WHERE c.deleted_at IS NULL AND c.status IN ('completed', 'terminated')
				 AND date_trunc('month', c.actual_sign_off_date) = m.m)
			FROM months m ORDER BY m.m`, today)
		if err != nil {
			return err
		}
		ch.Movements, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.MonthlyMovement, error) {
			var m model.MonthlyMovement
			err := row.Scan(&m.Month, &m.SignOns, &m.SignOffs)
			return m, err
		})
		return err
	})
	if err := g.Wait(); err != nil {
		return ch, fmt.Errorf("dashboard charts: %w", err)
	}
	return ch, nil
}

// Expiring lists the latest document or certificate per type expiring within days of today (or already expired), nearest first.
func (r *DashboardRepository) Expiring(ctx context.Context, today time.Time, days, limit int) ([]model.ExpiringItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT x.kind, c.public_id, c.full_name, coalesce(rk.name, ''), x.type_name, x.number, x.expired_date
		FROM (`+latestExpiryPerType+`) x
		JOIN crews c ON c.id = x.crew_id AND c.deleted_at IS NULL
		LEFT JOIN ranks rk ON rk.id = coalesce(
			(SELECT ac.rank_id FROM contracts ac WHERE ac.crew_id = c.id AND ac.status = 'active' AND ac.deleted_at IS NULL), c.applied_rank_id)
		WHERE x.expired_date <= $1::date + $2::int
		ORDER BY x.expired_date, c.full_name LIMIT $3`, today, days, limit)
	if err != nil {
		return nil, fmt.Errorf("dashboard expiring: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.ExpiringItem, error) {
		var it model.ExpiringItem
		err := row.Scan(&it.Kind, &it.CrewID, &it.CrewName, &it.RankName, &it.TypeName, &it.Number, &it.ExpiredDate)
		return it, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan dashboard expiring: %w", err)
	}
	return out, nil
}

// Reliefs lists active contracts due for relief within days of today, by planned relief date (or planned sign-off).
func (r *DashboardRepository) Reliefs(ctx context.Context, today time.Time, days, limit int) ([]model.UpcomingRelief, error) {
	rows, err := r.pool.Query(ctx, `SELECT c.public_id, cr.public_id, cr.full_name, rk.name, v.name, p.name, c.planned_sign_off_date,
			rp.id, rp.planned_relief_date, coalesce(rp.status, ''), coalesce(rl.public_id, ''), coalesce(rl.full_name, '')
		FROM contracts c
		JOIN crews cr ON cr.id = c.crew_id AND cr.deleted_at IS NULL
		JOIN ranks rk ON rk.id = c.rank_id
		JOIN vessels v ON v.id = c.vessel_id
		JOIN companies p ON p.id = c.principal_id
		LEFT JOIN LATERAL (SELECT * FROM relief_plans x WHERE x.contract_id = c.id AND x.status IN ('planned', 'confirmed')
			ORDER BY x.planned_relief_date LIMIT 1) rp ON true
		LEFT JOIN crews rl ON rl.id = rp.reliever_crew_id
		WHERE c.status = 'active' AND c.deleted_at IS NULL
		  AND coalesce(rp.planned_relief_date, c.planned_sign_off_date) <= $1::date + $2::int
		ORDER BY coalesce(rp.planned_relief_date, c.planned_sign_off_date), cr.full_name LIMIT $3`, today, days, limit)
	if err != nil {
		return nil, fmt.Errorf("dashboard reliefs: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.UpcomingRelief, error) {
		var u model.UpcomingRelief
		err := row.Scan(&u.ContractID, &u.CrewID, &u.CrewName, &u.RankName, &u.VesselName, &u.PrincipalName, &u.PlannedSignOff,
			&u.ReliefPlanID, &u.PlannedReliefDate, &u.ReliefStatus, &u.RelieverID, &u.RelieverName)
		return u, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan dashboard reliefs: %w", err)
	}
	return out, nil
}
