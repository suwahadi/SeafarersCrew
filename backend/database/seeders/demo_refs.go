package seeders

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type demoRank struct {
	ID         uuid.UUID
	Name       string
	Department string
	IsOfficer  bool
}

type demoType struct {
	ID        uuid.UUID
	Name      string
	HasExpiry bool
}

type demoAllowance struct {
	ID    uuid.UUID
	Basis string
}

type demoRefs struct {
	ranks          map[string]demoRank
	vesselTypes    map[string]uuid.UUID
	documentTypes  map[string]demoType
	certificates   map[string]demoType
	allowanceTypes map[string]demoAllowance
}

type codeRow[T any] struct {
	key   string
	value T
}

func collectRefs[T any](ctx context.Context, pool *pgxpool.Pool, sql string, scan func(pgx.CollectableRow) (codeRow[T], error)) (map[string]T, error) {
	rows, err := pool.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("query reference data: %w", err)
	}
	items, err := pgx.CollectRows(rows, scan)
	if err != nil {
		return nil, fmt.Errorf("scan reference data: %w", err)
	}
	out := make(map[string]T, len(items))
	for _, it := range items {
		out[it.key] = it.value
	}
	return out, nil
}

func loadDemoRefs(ctx context.Context, pool *pgxpool.Pool) (demoRefs, error) {
	var refs demoRefs
	var err error
	if refs.ranks, err = collectRefs(ctx, pool, `SELECT upper(short_code), id, name, department, is_officer FROM ranks WHERE deleted_at IS NULL`,
		func(r pgx.CollectableRow) (codeRow[demoRank], error) {
			var c codeRow[demoRank]
			err := r.Scan(&c.key, &c.value.ID, &c.value.Name, &c.value.Department, &c.value.IsOfficer)
			return c, err
		}); err != nil {
		return refs, err
	}
	if refs.vesselTypes, err = collectRefs(ctx, pool, `SELECT name, id FROM vessel_types WHERE deleted_at IS NULL`,
		func(r pgx.CollectableRow) (codeRow[uuid.UUID], error) {
			var c codeRow[uuid.UUID]
			err := r.Scan(&c.key, &c.value)
			return c, err
		}); err != nil {
		return refs, err
	}
	typeScan := func(r pgx.CollectableRow) (codeRow[demoType], error) {
		var c codeRow[demoType]
		err := r.Scan(&c.key, &c.value.ID, &c.value.Name, &c.value.HasExpiry)
		return c, err
	}
	if refs.documentTypes, err = collectRefs(ctx, pool, `SELECT code, id, name, has_expiry FROM document_types WHERE deleted_at IS NULL`, typeScan); err != nil {
		return refs, err
	}
	if refs.certificates, err = collectRefs(ctx, pool, `SELECT code, id, name, has_expiry FROM certificate_types WHERE deleted_at IS NULL`, typeScan); err != nil {
		return refs, err
	}
	if refs.allowanceTypes, err = collectRefs(ctx, pool, `SELECT name, id, basis FROM allowance_types WHERE deleted_at IS NULL`,
		func(r pgx.CollectableRow) (codeRow[demoAllowance], error) {
			var c codeRow[demoAllowance]
			err := r.Scan(&c.key, &c.value.ID, &c.value.Basis)
			return c, err
		}); err != nil {
		return refs, err
	}
	return refs, refs.validate()
}

func (r demoRefs) validate() error {
	for _, rank := range []string{"MST", "CO", "2O", "3O", "BSN", "AB", "OS", "CE", "2E", "3E", "4E", "EC", "OLR", "WPR", "CK", "STW"} {
		if _, ok := r.ranks[rank]; !ok {
			return fmt.Errorf("rank %s is missing; run the reference seed first", rank)
		}
	}
	for _, t := range []string{"AHTS", "Crew Boat", "ASD Tug", "Tanker", "Bulk Carrier", "RORO"} {
		if _, ok := r.vesselTypes[t]; !ok {
			return fmt.Errorf("vessel type %s is missing; run the reference seed first", t)
		}
	}
	for _, c := range []string{"passport", "seaman_book", "mcu", "ktp", "buku_pelaut", "yellow_fever", "visa"} {
		if _, ok := r.documentTypes[c]; !ok {
			return fmt.Errorf("document type %s is missing; run the reference seed first", c)
		}
	}
	for _, c := range []string{"bst", "sat", "aff", "mfa", "pscrb", "rating_nav_watch", "rating_eng_watch", "gmdss", "bosiet", "huet"} {
		if _, ok := r.certificates[c]; !ok {
			return fmt.Errorf("certificate type %s is missing; run the reference seed first", c)
		}
	}
	for _, a := range []string{"Overtime", "Leave Pay", "Standby Pay", "Offshore Allowance", "Bonus"} {
		if _, ok := r.allowanceTypes[a]; !ok {
			return fmt.Errorf("allowance type %s is missing; run the reference seed first", a)
		}
	}
	return nil
}
