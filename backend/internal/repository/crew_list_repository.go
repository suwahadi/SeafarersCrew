package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

const crewListFrom = `
FROM crews c
JOIN crew_statuses s ON s.id = c.crew_status_id
LEFT JOIN contracts ac ON ac.crew_id = c.id AND ac.status = 'active' AND ac.deleted_at IS NULL
LEFT JOIN ranks er ON er.id = coalesce(ac.rank_id, c.applied_rank_id)
LEFT JOIN vessels v ON v.id = ac.vessel_id
LEFT JOIN companies p ON p.id = ac.principal_id
LEFT JOIN companies ec ON ec.id = ac.employer_company_id`

const latestExpiries = `(
	SELECT max(td.expired_date) AS e FROM travel_documents td JOIN document_types dt ON dt.id = td.document_type_id
	WHERE td.crew_id = c.id AND dt.has_expiry AND dt.deleted_at IS NULL GROUP BY td.document_type_id
	UNION ALL
	SELECT max(ce.expired_date) FROM certificates ce JOIN certificate_types ct ON ct.id = ce.certificate_type_id
	WHERE ce.crew_id = c.id AND ct.has_expiry AND ct.deleted_at IS NULL GROUP BY ce.certificate_type_id
)`

func compliantExpr(today string) string {
	return `(er.id IS NOT NULL AND NOT EXISTS (
	SELECT 1 FROM compliance_requirements cr
	LEFT JOIN document_types rdt ON rdt.id = cr.document_type_id
	LEFT JOIN certificate_types rct ON rct.id = cr.certificate_type_id
	WHERE cr.rank_id = er.id AND coalesce(rdt.deleted_at, rct.deleted_at) IS NULL
	AND NOT EXISTS (
		SELECT 1 FROM travel_documents td
		WHERE cr.document_type_id IS NOT NULL AND td.crew_id = c.id AND td.document_type_id = cr.document_type_id
		AND (NOT rdt.has_expiry OR td.expired_date >= ` + today + `)
		UNION ALL
		SELECT 1 FROM certificates ce
		WHERE cr.certificate_type_id IS NOT NULL AND ce.crew_id = c.id AND ce.certificate_type_id = cr.certificate_type_id
		AND (NOT rct.has_expiry OR ce.expired_date >= ` + today + `)
	)))`
}

func seaDaysExpr(today, rankFilter string) string {
	return `(SELECT coalesce(sum(upper(r) - lower(r)), 0) FROM unnest((
	SELECT range_agg(daterange(e.sign_on_date, greatest(e.sign_on_date, coalesce(e.sign_off_date, ` + today + `)), '[]'))
	FROM experiences e WHERE e.crew_id = c.id` + rankFilter + `)) r)`
}

func nearestExpiryExpr() string {
	return `(SELECT min(x.e) FROM ` + latestExpiries + ` x)`
}

func groupExprs(groupBy string) (key, name string) {
	switch groupBy {
	case "principal":
		return "coalesce(p.id::text, '')", "coalesce(p.name, '')"
	case "company":
		return "coalesce(ec.id::text, '')", "coalesce(ec.name, '')"
	}
	return "", ""
}

func crewWhere(f model.CrewFilter) (*Where, func() string) {
	w := &Where{}
	var todayRef string
	today := func() string {
		if todayRef == "" {
			todayRef = w.Arg(f.Today) + "::date"
		}
		return todayRef
	}
	w.Add("c.deleted_at IS NULL")
	if f.Q != "" {
		like := LikePattern(f.Q)
		w.Add(`(c.full_name ILIKE %s OR c.public_id = %s OR EXISTS (
			SELECT 1 FROM travel_documents td JOIN document_types dt ON dt.id = td.document_type_id
			WHERE td.crew_id = c.id AND dt.code IN ('passport', 'seaman_book', 'buku_pelaut') AND td.number ILIKE %s))`,
			like, strings.ToUpper(f.Q), like)
	}
	if len(f.StatusCodes) > 0 {
		w.Add("s.code = ANY(%s)", f.StatusCodes)
	}
	if len(f.ExcludeStatuses) > 0 {
		w.Add("NOT (coalesce(s.code, '') = ANY(%s))", f.ExcludeStatuses)
	}
	if len(f.RankIDs) > 0 {
		w.Add("er.id = ANY(%s)", f.RankIDs)
	}
	if len(f.Departments) > 0 {
		w.Add("er.department = ANY(%s)", f.Departments)
	}
	if len(f.VesselTypeIDs) > 0 {
		w.Add("v.vessel_type_id = ANY(%s)", f.VesselTypeIDs)
	}
	if len(f.VesselIDs) > 0 {
		w.Add("ac.vessel_id = ANY(%s)", f.VesselIDs)
	}
	if len(f.PrincipalIDs) > 0 {
		w.Add("ac.principal_id = ANY(%s)", f.PrincipalIDs)
	}
	if f.AgeMin != nil {
		w.Add("c.date_of_birth <= "+today()+" - make_interval(years => %s)", *f.AgeMin)
	}
	if f.AgeMax != nil {
		w.Add("c.date_of_birth > "+today()+" - make_interval(years => %s)", *f.AgeMax+1)
	}
	if len(f.Nationalities) > 0 {
		lower := make([]string, len(f.Nationalities))
		for i, n := range f.Nationalities {
			lower[i] = strings.ToLower(n)
		}
		w.Add("lower(c.nationality) = ANY(%s)", lower)
	}
	switch f.Compliance {
	case "compliant":
		w.Add(compliantExpr(today()))
	case "not_compliant":
		w.Add("NOT " + compliantExpr(today()))
	}
	days := f.DocDays
	if days <= 0 {
		days = 90
	}
	switch f.DocValidity {
	case "expired":
		w.Add(`EXISTS (SELECT 1 FROM ` + latestExpiries + ` x WHERE x.e < ` + today() + `)`)
	case "expiring":
		w.Add(`EXISTS (SELECT 1 FROM `+latestExpiries+` x WHERE x.e >= `+today()+` AND x.e <= `+today()+` + %s::int)`, days)
	case "valid":
		w.Add(`NOT EXISTS (SELECT 1 FROM `+latestExpiries+` x WHERE x.e <= `+today()+` + %s::int)`, days)
	}
	if f.SeaRankID != nil && f.SeaMinDays > 0 {
		rank := w.Arg(*f.SeaRankID)
		w.Add(seaDaysExpr(today(), " AND e.rank_id = "+rank)+" >= %s", f.SeaMinDays)
	}
	return w, today
}

func crewOrder(f model.CrewFilter, today func() string) string {
	cols := map[string]string{
		"id": "c.public_id", "full_name": "c.full_name", "nationality": "c.nationality", "date_of_birth": "c.date_of_birth",
		"age": "c.date_of_birth", "status": "s.sort_order", "rank": "er.sort_order", "department": "er.department",
		"mobile_phone": "c.mobile_phone", "email": "c.email", "vessel": "v.name", "principal": "p.name",
		"created_at": "c.created_at", "nearest_expiry": nearestExpiryExpr(),
	}
	switch f.Sort {
	case "sea_service":
		cols["sea_service"] = seaDaysExpr(today(), "")
	case "compliant":
		cols["compliant"] = compliantExpr(today())
	}
	desc := f.Desc
	if f.Sort == "age" {
		desc = !desc
	}
	order := OrderWithTie(f.Sort, desc, cols, "c.created_at DESC, c.id DESC", "c.id")
	if _, name := groupExprs(f.GroupBy); name != "" {
		order = " ORDER BY " + name + " ASC," + strings.TrimPrefix(order, " ORDER BY")
	}
	return order
}

// ListCrews returns one datagrid page with its total and, when grouping, the group counts.
func (r *CrewRepository) ListCrews(ctx context.Context, f model.CrewFilter, limit, offset int) (Page[model.CrewRow], []model.CrewGroup, error) {
	var (
		page   Page[model.CrewRow]
		groups []model.CrewGroup
		ids    []uuid.UUID
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		w, _ := crewWhere(f)
		return r.pool.QueryRow(gctx, `SELECT count(*)`+crewListFrom+w.SQL(), w.Args()...).Scan(&page.Total)
	})
	g.Go(func() error {
		w, today := crewWhere(f)
		lim, off := w.Arg(limit), w.Arg(offset)
		rows, err := r.pool.Query(gctx, `SELECT c.id`+crewListFrom+w.SQL()+crewOrder(f, today)+` LIMIT `+lim+` OFFSET `+off, w.Args()...)
		if err != nil {
			return err
		}
		ids, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		return err
	})
	if key, name := groupExprs(f.GroupBy); key != "" {
		g.Go(func() error {
			w, _ := crewWhere(f)
			rows, err := r.pool.Query(gctx, `SELECT `+key+`, `+name+`, count(*)`+crewListFrom+w.SQL()+` GROUP BY 1, 2 ORDER BY 2`, w.Args()...)
			if err != nil {
				return err
			}
			groups, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.CrewGroup, error) {
				var gr model.CrewGroup
				err := row.Scan(&gr.Key, &gr.Name, &gr.Count)
				return gr, err
			})
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return page, nil, fmt.Errorf("list crews: %w", err)
	}
	items, err := r.crewRows(ctx, ids, f)
	if err != nil {
		return page, nil, err
	}
	page.Items = items
	return page, groups, nil
}

// StreamCrews calls fn for every row matching f in display order, fetching in batches.
func (r *CrewRepository) StreamCrews(ctx context.Context, f model.CrewFilter, batch int, fn func(model.CrewRow) error) error {
	for offset := 0; ; offset += batch {
		w, today := crewWhere(f)
		lim, off := w.Arg(batch), w.Arg(offset)
		rows, err := r.pool.Query(ctx, `SELECT c.id`+crewListFrom+w.SQL()+crewOrder(f, today)+` LIMIT `+lim+` OFFSET `+off, w.Args()...)
		if err != nil {
			return fmt.Errorf("stream crews: %w", err)
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if err != nil {
			return fmt.Errorf("stream crews: %w", err)
		}
		items, err := r.crewRows(ctx, ids, f)
		if err != nil {
			return err
		}
		for _, it := range items {
			if err := fn(it); err != nil {
				return err
			}
		}
		if len(ids) < batch {
			return nil
		}
	}
}

func (r *CrewRepository) crewRows(ctx context.Context, ids []uuid.UUID, f model.CrewFilter) ([]model.CrewRow, error) {
	if len(ids) == 0 {
		return []model.CrewRow{}, nil
	}
	key, _ := groupExprs(f.GroupBy)
	if key == "" {
		key = "''"
	}
	rows, err := r.pool.Query(ctx, `SELECT c.public_id, c.full_name, c.date_of_birth, c.nationality, c.mobile_phone, c.email,
		coalesce(s.code, ''), s.name, s.color, er.id, coalesce(er.name, ''), coalesce(er.department, ''), coalesce(ac.public_id, ''),
		v.id, coalesce(v.name, ''), coalesce(vt.name, ''), p.id, coalesce(p.name, ''), coalesce(ec.name, ''),
		coalesce((SELECT td.number FROM travel_documents td JOIN document_types dt ON dt.id = td.document_type_id
			WHERE td.crew_id = c.id AND dt.code = 'passport' ORDER BY td.expired_date DESC NULLS LAST LIMIT 1), ''),
		coalesce((SELECT td.number FROM travel_documents td JOIN document_types dt ON dt.id = td.document_type_id
			WHERE td.crew_id = c.id AND dt.code IN ('seaman_book', 'buku_pelaut') ORDER BY td.expired_date DESC NULLS LAST LIMIT 1), ''),
		`+compliantExpr("$2::date")+`, `+nearestExpiryExpr()+`, `+seaDaysExpr("$2::date", "")+`, `+key+`, c.created_at
		FROM unnest($1::uuid[]) WITH ORDINALITY o(id, ord)
		JOIN crews c ON c.id = o.id
		JOIN crew_statuses s ON s.id = c.crew_status_id
		LEFT JOIN contracts ac ON ac.crew_id = c.id AND ac.status = 'active' AND ac.deleted_at IS NULL
		LEFT JOIN ranks er ON er.id = coalesce(ac.rank_id, c.applied_rank_id)
		LEFT JOIN vessels v ON v.id = ac.vessel_id
		LEFT JOIN vessel_types vt ON vt.id = v.vessel_type_id
		LEFT JOIN companies p ON p.id = ac.principal_id
		LEFT JOIN companies ec ON ec.id = ac.employer_company_id
		ORDER BY o.ord`, ids, f.Today)
	if err != nil {
		return nil, fmt.Errorf("load crew rows: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.CrewRow, error) {
		var c model.CrewRow
		err := row.Scan(&c.PublicID, &c.FullName, &c.DateOfBirth, &c.Nationality, &c.MobilePhone, &c.Email,
			&c.StatusCode, &c.StatusName, &c.StatusColor, &c.RankID, &c.RankName, &c.Department, &c.ContractID,
			&c.VesselID, &c.VesselName, &c.VesselType, &c.PrincipalID, &c.PrincipalName, &c.EmployerName,
			&c.PassportNumber, &c.SeamanBook, &c.Compliant, &c.NearestExpiry, &c.SeaServiceDays, &c.GroupKey, &c.CreatedAt)
		return c, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan crew rows: %w", err)
	}
	return out, nil
}
