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
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
)

// Alias kinds stored in master_aliases.
const (
	CVKindCertificateType = "certificate_type"
	CVKindDocumentType    = "document_type"
	CVKindRank            = "rank"
	CVKindVesselType      = "vessel_type"
)

// ErrCVDuplicate means the crew member of a CV is already registered.
var ErrCVDuplicate = errors.New("crew already registered")

// CVMasterItem is one active master record used for CV matching.
type CVMasterItem struct {
	ID         uuid.UUID
	Code, Name string
	HasExpiry  bool
}

// CVLookups holds the master data and aliases a CV import matches against.
type CVLookups struct {
	CertificateTypes, DocumentTypes, Ranks, VesselTypes, Companies []CVMasterItem
	Aliases                                                        map[string]map[string]uuid.UUID
	PassportType, SeamanBookType                                   uuid.UUID
}

// CVDuplicate is an existing crew member that a CV matches.
type CVDuplicate struct {
	PublicID string `json:"crew_id"`
	FullName string `json:"full_name"`
	Reason   string `json:"reason"`
}

// CVDuplicateKeys are the values a CV is checked against for duplicates.
type CVDuplicateKeys struct {
	PassportNumber, SeamanBookNumber string
	FullName                         string
	DateOfBirth                      *time.Time
	PassportType, SeamanBookType     uuid.UUID
}

// CVNewType is a master type the admin chose to create during a CV import.
type CVNewType struct {
	Key, Kind, Name, Code, Category string
	HasExpiry                       bool
}

// CVAlias remembers that a CV text means an existing master record.
type CVAlias struct {
	Kind, Alias string
	TargetID    uuid.UUID
}

// CVRecord is a travel document or certificate whose type is known or created in the same import.
type CVRecord struct {
	TypeID     uuid.UUID
	NewTypeKey string
	Doc        model.CrewDocument
}

// CVSea is a sea-service row whose vessel type may be created in the same import.
type CVSea struct {
	VesselTypeKey string
	Experience    model.Experience
}

// CVCreate is everything one CV import writes.
type CVCreate struct {
	Crew         model.Crew
	Keys         CVDuplicateKeys
	NewTypes     []CVNewType
	Aliases      []CVAlias
	Kins         []model.NextOfKin
	Documents    []CVRecord
	Certificates []CVRecord
	SeaService   []CVSea
}

// CrewCVRepository reads lookups for and writes the result of a CV import.
type CrewCVRepository struct{ pool *pgxpool.Pool }

// NewCrewCVRepository builds a CrewCVRepository.
func NewCrewCVRepository(pool *pgxpool.Pool) *CrewCVRepository { return &CrewCVRepository{pool: pool} }

func collectCVItems(ctx context.Context, db DBTX, sql string) ([]CVMasterItem, error) {
	rows, err := db.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (CVMasterItem, error) {
		var it CVMasterItem
		err := row.Scan(&it.ID, &it.Code, &it.Name, &it.HasExpiry)
		return it, err
	})
}

// Lookups loads active master data and the aliases whose target is still live.
func (r *CrewCVRepository) Lookups(ctx context.Context) (CVLookups, error) {
	l := CVLookups{Aliases: map[string]map[string]uuid.UUID{}}
	g, gctx := errgroup.WithContext(ctx)
	load := func(dst *[]CVMasterItem, sql string) {
		g.Go(func() error {
			items, err := collectCVItems(gctx, r.pool, sql)
			if err != nil {
				return fmt.Errorf("load cv lookups: %w", err)
			}
			*dst = items
			return nil
		})
	}
	load(&l.CertificateTypes, `SELECT id, coalesce(code, ''), name, has_expiry FROM certificate_types WHERE deleted_at IS NULL AND is_active ORDER BY sort_order, name`)
	load(&l.DocumentTypes, `SELECT id, coalesce(code, ''), name, has_expiry FROM document_types WHERE deleted_at IS NULL AND is_active ORDER BY sort_order, name`)
	load(&l.Ranks, `SELECT id, short_code, name, false FROM ranks WHERE deleted_at IS NULL AND is_active ORDER BY sort_order, name`)
	load(&l.VesselTypes, `SELECT id, '', name, false FROM vessel_types WHERE deleted_at IS NULL AND is_active ORDER BY sort_order, name`)
	load(&l.Companies, `SELECT id, coalesce(code, ''), name, false FROM companies WHERE deleted_at IS NULL AND is_active ORDER BY name`)
	g.Go(func() error {
		rows, err := r.pool.Query(gctx, `SELECT a.kind, a.alias, a.target_id FROM master_aliases a
			WHERE (a.kind = 'certificate_type' AND EXISTS (SELECT 1 FROM certificate_types t WHERE t.id = a.target_id AND t.deleted_at IS NULL))
				OR (a.kind = 'document_type' AND EXISTS (SELECT 1 FROM document_types t WHERE t.id = a.target_id AND t.deleted_at IS NULL))
				OR (a.kind = 'rank' AND EXISTS (SELECT 1 FROM ranks t WHERE t.id = a.target_id AND t.deleted_at IS NULL))
				OR (a.kind = 'vessel_type' AND EXISTS (SELECT 1 FROM vessel_types t WHERE t.id = a.target_id AND t.deleted_at IS NULL))`)
		if err != nil {
			return fmt.Errorf("load cv aliases: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var kind, alias string
			var id uuid.UUID
			if err := rows.Scan(&kind, &alias, &id); err != nil {
				return fmt.Errorf("scan cv alias: %w", err)
			}
			if l.Aliases[kind] == nil {
				l.Aliases[kind] = map[string]uuid.UUID{}
			}
			l.Aliases[kind][alias] = id
		}
		return rows.Err()
	})
	if err := g.Wait(); err != nil {
		return CVLookups{}, err
	}
	for _, t := range l.DocumentTypes {
		switch t.Code {
		case "passport":
			l.PassportType = t.ID
		case "seaman_book":
			l.SeamanBookType = t.ID
		}
	}
	return l, nil
}

// Duplicates returns live crew members matching the CV's passport, seaman book, or name and birth date.
func (r *CrewCVRepository) Duplicates(ctx context.Context, k CVDuplicateKeys) ([]CVDuplicate, error) {
	return cvDuplicates(ctx, r.pool, k)
}

func cvDuplicates(ctx context.Context, db DBTX, k CVDuplicateKeys) ([]CVDuplicate, error) {
	rows, err := db.Query(ctx, `SELECT c.public_id, c.full_name, 'passport' FROM travel_documents d JOIN crews c ON c.id = d.crew_id
			WHERE $1 <> '' AND d.document_type_id = $2 AND c.deleted_at IS NULL AND upper(replace(d.number, ' ', '')) = upper(replace($1, ' ', ''))
		UNION ALL
		SELECT c.public_id, c.full_name, 'seaman_book' FROM travel_documents d JOIN crews c ON c.id = d.crew_id
			WHERE $3 <> '' AND d.document_type_id = $4 AND c.deleted_at IS NULL AND upper(replace(d.number, ' ', '')) = upper(replace($3, ' ', ''))
		UNION ALL
		SELECT c.public_id, c.full_name, 'name_and_birth_date' FROM crews c
			WHERE $5 <> '' AND $6::date IS NOT NULL AND c.deleted_at IS NULL AND lower(c.full_name) = lower($5) AND c.date_of_birth = $6::date`,
		k.PassportNumber, k.PassportType, k.SeamanBookNumber, k.SeamanBookType, k.FullName, k.DateOfBirth)
	if err != nil {
		return nil, fmt.Errorf("find cv duplicates: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (CVDuplicate, error) {
		var d CVDuplicate
		err := row.Scan(&d.PublicID, &d.FullName, &d.Reason)
		return d, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan cv duplicates: %w", err)
	}
	return out, nil
}

// Create writes the crew, its records, new types and aliases in one transaction, refusing duplicates.
func (r *CrewCVRepository) Create(ctx context.Context, in CVCreate, newID func() (string, error), attempts int) (string, error) {
	var pid string
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('crew_cv_import'))`); err != nil {
			return fmt.Errorf("lock cv import: %w", err)
		}
		dups, err := cvDuplicates(ctx, tx, in.Keys)
		if err != nil {
			return err
		}
		if len(dups) > 0 {
			return ErrCVDuplicate
		}
		created := map[string]uuid.UUID{}
		for _, t := range in.NewTypes {
			id, err := insertCVType(ctx, tx, t)
			if err != nil {
				return err
			}
			created[t.Key] = id
		}
		for _, a := range in.Aliases {
			if _, err := tx.Exec(ctx, `INSERT INTO master_aliases (kind, alias, target_id) VALUES ($1, $2, $3)
				ON CONFLICT (kind, alias) DO UPDATE SET target_id = EXCLUDED.target_id`, a.Kind, a.Alias, a.TargetID); err != nil {
				return fmt.Errorf("save cv alias: %w", err)
			}
		}
		var crewID uuid.UUID
		crewID, pid, err = insertCVCrew(ctx, tx, in.Crew, newID, attempts)
		if err != nil {
			return err
		}
		return insertCVRecords(ctx, tx, crewID, in, created)
	})
	if err != nil {
		return "", err
	}
	return pid, nil
}

func cvTypeID(id uuid.UUID, key string, created map[string]uuid.UUID) (uuid.UUID, error) {
	if key == "" {
		return id, nil
	}
	if made, ok := created[key]; ok {
		return made, nil
	}
	return uuid.Nil, fmt.Errorf("cv import references an unknown new type")
}

func insertCVRecords(ctx context.Context, tx pgx.Tx, crewID uuid.UUID, in CVCreate, created map[string]uuid.UUID) error {
	for _, k := range in.Kins {
		if _, err := tx.Exec(ctx, `INSERT INTO crew_next_of_kins (crew_id, name, relationship, phone, address, is_primary) VALUES ($1, $2, $3, $4, $5, $6)`,
			crewID, k.Name, k.Relationship, k.Phone, k.Address, k.IsPrimary); err != nil {
			return fmt.Errorf("insert cv next of kin: %w", err)
		}
	}
	for _, d := range in.Documents {
		typeID, err := cvTypeID(d.TypeID, d.NewTypeKey, created)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO travel_documents (crew_id, document_type_id, number, issued_date, expired_date, place_of_issue)
			VALUES ($1, $2, $3, $4, $5, $6)`, crewID, typeID, d.Doc.Number, d.Doc.IssuedDate, d.Doc.ExpiredDate, d.Doc.PlaceOfIssue); err != nil {
			return fmt.Errorf("insert cv travel document: %w", err)
		}
	}
	for _, c := range in.Certificates {
		typeID, err := cvTypeID(c.TypeID, c.NewTypeKey, created)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO certificates (crew_id, certificate_type_id, number, issued_date, expired_date, issuing_authority, place_of_issue)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`, crewID, typeID, c.Doc.Number, c.Doc.IssuedDate, c.Doc.ExpiredDate, c.Doc.IssuingAuthority, c.Doc.PlaceOfIssue); err != nil {
			return fmt.Errorf("insert cv certificate: %w", err)
		}
	}
	for _, s := range in.SeaService {
		x := s.Experience
		if s.VesselTypeKey != "" {
			id, err := cvTypeID(uuid.Nil, s.VesselTypeKey, created)
			if err != nil {
				return err
			}
			x.VesselTypeID = &id
		}
		if _, err := tx.Exec(ctx, `INSERT INTO experiences (crew_id, vessel_name, rank_id, vessel_type_id, grt_hp, company_id, company_name,
			sign_on_date, sign_off_date, status, remarks, source) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, coalesce(nullif($10, ''), 'sign_off'), $11, 'manual')`,
			crewID, x.VesselName, x.RankID, x.VesselTypeID, x.GRTHP, x.CompanyID, x.CompanyName, x.SignOnDate, x.SignOffDate, x.Status, x.Remarks); err != nil {
			return fmt.Errorf("insert cv sea service: %w", err)
		}
	}
	return nil
}

func insertCVType(ctx context.Context, tx pgx.Tx, t CVNewType) (uuid.UUID, error) {
	table := map[string]string{CVKindCertificateType: "certificate_types", CVKindDocumentType: "document_types", CVKindVesselType: "vessel_types"}[t.Kind]
	if table == "" {
		return uuid.Nil, fmt.Errorf("cv import cannot create a %q", t.Kind)
	}
	for attempt := 1; attempt <= 9; attempt++ {
		code := t.Code
		if attempt > 1 {
			code = fmt.Sprintf("%s_%d", t.Code, attempt)
		}
		var id uuid.UUID
		var err error
		switch t.Kind {
		case CVKindCertificateType:
			err = tx.QueryRow(ctx, `INSERT INTO certificate_types (code, name, stcw_reference, category, has_expiry) VALUES ($1, $2, '', $3, $4)
				ON CONFLICT DO NOTHING RETURNING id`, code, t.Name, t.Category, t.HasExpiry).Scan(&id)
		case CVKindDocumentType:
			err = tx.QueryRow(ctx, `INSERT INTO document_types (code, name, has_expiry) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING RETURNING id`,
				code, t.Name, t.HasExpiry).Scan(&id)
		default:
			err = tx.QueryRow(ctx, `INSERT INTO vessel_types (name) VALUES ($1) ON CONFLICT DO NOTHING RETURNING id`, t.Name).Scan(&id)
		}
		if err == nil {
			return id, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("insert cv %s: %w", t.Kind, err)
		}
		err = tx.QueryRow(ctx, `SELECT id FROM `+table+` WHERE lower(name) = lower($1) AND deleted_at IS NULL`, t.Name).Scan(&id)
		if err == nil {
			return id, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("find cv %s: %w", t.Kind, err)
		}
	}
	return uuid.Nil, fmt.Errorf("no free code for the new %s", t.Kind)
}

func insertCVCrew(ctx context.Context, tx pgx.Tx, c model.Crew, newID func() (string, error), attempts int) (uuid.UUID, string, error) {
	for range attempts {
		pid, err := newID()
		if err != nil {
			return uuid.Nil, "", err
		}
		sp, err := tx.Begin(ctx)
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("savepoint: %w", err)
		}
		var id uuid.UUID
		err = sp.QueryRow(ctx, `INSERT INTO crews (public_id, full_name, place_of_birth, date_of_birth, gender, nationality, religion,
			marital_status, coverall_size, applied_rank_id, crew_status_id, cv_file_id, address, home_phone, mobile_phone, email)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, s.id, $11, $12, $13, $14, $15 FROM crew_statuses s WHERE s.code = 'available'
			RETURNING id`,
			pid, c.FullName, c.PlaceOfBirth, c.DateOfBirth, c.Gender, c.Nationality, c.Religion, c.MaritalStatus, c.CoverallSize,
			c.AppliedRankID, c.CVFileID, c.Address, c.HomePhone, c.MobilePhone, c.Email).Scan(&id)
		if err == nil {
			if err := sp.Commit(ctx); err != nil {
				return uuid.Nil, "", fmt.Errorf("release savepoint: %w", err)
			}
			return id, pid, nil
		}
		if rbErr := sp.Rollback(ctx); rbErr != nil {
			return uuid.Nil, "", fmt.Errorf("rollback savepoint: %w", rbErr)
		}
		if !postgres.IsUniqueViolation(err, "crews_public_id_key") {
			return uuid.Nil, "", fmt.Errorf("insert cv crew: %w", notFound(err))
		}
	}
	return uuid.Nil, "", fmt.Errorf("no unique crew id after %d attempts", attempts)
}
