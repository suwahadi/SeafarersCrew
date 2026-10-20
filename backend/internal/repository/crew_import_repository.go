package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
)

// ImportLookups resolves names used in import files.
type ImportLookups struct {
	Ranks        map[string]uuid.UUID
	PassportType uuid.UUID
	SeamanBook   uuid.UUID
}

// ImportLookups loads active ranks by short code and name, and the passport and seaman book types.
func (r *CrewRepository) ImportLookups(ctx context.Context) (ImportLookups, error) {
	l := ImportLookups{Ranks: map[string]uuid.UUID{}}
	rows, err := r.pool.Query(ctx, `SELECT id, short_code, name FROM ranks WHERE deleted_at IS NULL AND is_active`)
	if err != nil {
		return l, fmt.Errorf("load ranks: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var code, name string
		if err := rows.Scan(&id, &code, &name); err != nil {
			return l, fmt.Errorf("scan rank: %w", err)
		}
		l.Ranks[strings.ToUpper(code)] = id
		l.Ranks[strings.ToUpper(name)] = id
	}
	if err := rows.Err(); err != nil {
		return l, fmt.Errorf("load ranks: %w", err)
	}
	err = r.pool.QueryRow(ctx, `SELECT
		coalesce((SELECT id FROM document_types WHERE code = 'passport' AND deleted_at IS NULL), '00000000-0000-0000-0000-000000000000'),
		coalesce((SELECT id FROM document_types WHERE code = 'seaman_book' AND deleted_at IS NULL), '00000000-0000-0000-0000-000000000000')`).
		Scan(&l.PassportType, &l.SeamanBook)
	if err != nil {
		return l, fmt.Errorf("load document types: %w", err)
	}
	return l, nil
}

// ExistingDocumentNumbers returns which of numbers already exist for typeID, uppercased.
func (r *CrewRepository) ExistingDocumentNumbers(ctx context.Context, typeID uuid.UUID, numbers []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(numbers) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT upper(td.number) FROM travel_documents td JOIN crews c ON c.id = td.crew_id
		WHERE td.document_type_id = $1 AND c.deleted_at IS NULL AND upper(td.number) = ANY($2)`, typeID, numbers)
	if err != nil {
		return nil, fmt.Errorf("find existing documents: %w", err)
	}
	found, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("scan existing documents: %w", err)
	}
	for _, n := range found {
		out[n] = true
	}
	return out, nil
}

// ImportDocument is a document created with an imported crew member.
type ImportDocument struct {
	TypeID      uuid.UUID
	Number      string
	ExpiredDate *time.Time
}

// ImportCrew is one validated import row.
type ImportCrew struct {
	Line      int
	Crew      model.Crew
	Documents []ImportDocument
}

// ImportOutcome is the result of one import row.
type ImportOutcome struct {
	Line     int    `json:"line"`
	PublicID string `json:"crew_id,omitempty"`
	Error    string `json:"error,omitempty"`
}

// ImportCrews inserts rows in one transaction, isolating each row in a savepoint so a failed row does not stop the batch.
func (r *CrewRepository) ImportCrews(ctx context.Context, rows []ImportCrew, newID func() (string, error), attempts int) ([]ImportOutcome, error) {
	out := make([]ImportOutcome, 0, len(rows))
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		for _, row := range rows {
			pid, err := importRow(ctx, tx, row, newID, attempts)
			if err != nil {
				out = append(out, ImportOutcome{Line: row.Line, Error: err.Error()})
				continue
			}
			out = append(out, ImportOutcome{Line: row.Line, PublicID: pid})
		}
		return nil
	})
	return out, err
}

func importRow(ctx context.Context, tx pgx.Tx, row ImportCrew, newID func() (string, error), attempts int) (string, error) {
	for range attempts {
		pid, err := newID()
		if err != nil {
			return "", err
		}
		sp, err := tx.Begin(ctx)
		if err != nil {
			return "", fmt.Errorf("savepoint: %w", err)
		}
		err = insertImported(ctx, sp, row, pid)
		if err == nil {
			if err := sp.Commit(ctx); err != nil {
				return "", fmt.Errorf("release savepoint: %w", err)
			}
			return pid, nil
		}
		if rbErr := sp.Rollback(ctx); rbErr != nil {
			return "", fmt.Errorf("rollback savepoint: %w", rbErr)
		}
		if !postgres.IsUniqueViolation(err, "crews_public_id_key") {
			return "", fmt.Errorf("insert row: %w", err)
		}
	}
	return "", fmt.Errorf("no unique crew id after %d attempts", attempts)
}

func insertImported(ctx context.Context, tx pgx.Tx, row ImportCrew, pid string) error {
	c := row.Crew
	var id uuid.UUID
	err := tx.QueryRow(ctx, `INSERT INTO crews (public_id, full_name, place_of_birth, date_of_birth, gender, nationality, religion,
		marital_status, applied_rank_id, crew_status_id, address, city, province, country, postal_code, mobile_phone, email)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, s.id, $10, $11, $12, $13, $14, $15, $16
		FROM crew_statuses s WHERE s.code = 'available' RETURNING id`,
		pid, c.FullName, c.PlaceOfBirth, c.DateOfBirth, c.Gender, c.Nationality, c.Religion, c.MaritalStatus, c.AppliedRankID,
		c.Address, c.City, c.Province, c.Country, c.PostalCode, c.MobilePhone, c.Email).Scan(&id)
	if err != nil {
		return err
	}
	for _, d := range row.Documents {
		if _, err := tx.Exec(ctx, `INSERT INTO travel_documents (crew_id, document_type_id, number, expired_date) VALUES ($1, $2, $3, $4)`,
			id, d.TypeID, d.Number, d.ExpiredDate); err != nil {
			return err
		}
	}
	return nil
}
