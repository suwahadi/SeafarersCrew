package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// CrewRecordRepository stores the rows of the crew tabs: bank accounts, next of kin, documents, certificates and experiences.
type CrewRecordRepository struct {
	pool *pgxpool.Pool
}

// NewCrewRecordRepository returns a CrewRecordRepository.
func NewCrewRecordRepository(pool *pgxpool.Pool) *CrewRecordRepository {
	return &CrewRecordRepository{pool: pool}
}

func collect[T any](rows pgx.Rows, scan func(pgx.Row) (T, error), what string) ([]T, error) {
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (T, error) { return scan(row) })
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", what, err)
	}
	return out, nil
}

func affectedOne(tag interface{ RowsAffected() int64 }, err error, op string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

const bankColumns = `id, crew_id, bank_name, account_name, account_number, currency_code, cover_bank, remarks, is_primary`

func scanBank(row pgx.Row) (model.BankAccount, error) {
	var b model.BankAccount
	err := row.Scan(&b.ID, &b.CrewID, &b.BankName, &b.AccountName, &b.AccountNumber, &b.CurrencyCode, &b.CoverBank, &b.Remarks, &b.IsPrimary)
	return b, notFound(err)
}

// ListBankAccounts returns the bank accounts of a crew member, primary first.
func (r *CrewRecordRepository) ListBankAccounts(ctx context.Context, crewID uuid.UUID) ([]model.BankAccount, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+bankColumns+` FROM crew_bank_accounts WHERE crew_id = $1 ORDER BY is_primary DESC, created_at`, crewID)
	if err != nil {
		return nil, fmt.Errorf("list bank accounts: %w", err)
	}
	return collect(rows, scanBank, "bank accounts")
}

// SaveBankAccount inserts (zero ID) or updates a bank account, keeping a single primary per crew.
func (r *CrewRecordRepository) SaveBankAccount(ctx context.Context, b model.BankAccount) (model.BankAccount, error) {
	var out model.BankAccount
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		if err := clearPrimary(ctx, tx, "crew_bank_accounts", b.CrewID, b.ID, b.IsPrimary); err != nil {
			return err
		}
		var err error
		if b.ID == uuid.Nil {
			out, err = scanBank(tx.QueryRow(ctx, `INSERT INTO crew_bank_accounts (crew_id, bank_name, account_name, account_number,
				currency_code, cover_bank, remarks, is_primary) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING `+bankColumns,
				b.CrewID, b.BankName, b.AccountName, b.AccountNumber, b.CurrencyCode, b.CoverBank, b.Remarks, b.IsPrimary))
		} else {
			out, err = scanBank(tx.QueryRow(ctx, `UPDATE crew_bank_accounts SET bank_name = $3, account_name = $4, account_number = $5,
				currency_code = $6, cover_bank = $7, remarks = $8, is_primary = $9 WHERE id = $1 AND crew_id = $2 RETURNING `+bankColumns,
				b.ID, b.CrewID, b.BankName, b.AccountName, b.AccountNumber, b.CurrencyCode, b.CoverBank, b.Remarks, b.IsPrimary))
		}
		if err != nil {
			return fmt.Errorf("save bank account: %w", err)
		}
		return nil
	})
	return out, err
}

// DeleteBankAccount removes a bank account.
func (r *CrewRecordRepository) DeleteBankAccount(ctx context.Context, crewID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM crew_bank_accounts WHERE id = $1 AND crew_id = $2`, id, crewID)
	return affectedOne(tag, err, "delete bank account")
}

const kinColumns = `id, crew_id, name, relationship, phone, address, is_primary`

func scanKin(row pgx.Row) (model.NextOfKin, error) {
	var k model.NextOfKin
	err := row.Scan(&k.ID, &k.CrewID, &k.Name, &k.Relationship, &k.Phone, &k.Address, &k.IsPrimary)
	return k, notFound(err)
}

// ListNextOfKins returns the next of kin of a crew member, primary first.
func (r *CrewRecordRepository) ListNextOfKins(ctx context.Context, crewID uuid.UUID) ([]model.NextOfKin, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+kinColumns+` FROM crew_next_of_kins WHERE crew_id = $1 ORDER BY is_primary DESC, created_at`, crewID)
	if err != nil {
		return nil, fmt.Errorf("list next of kin: %w", err)
	}
	return collect(rows, scanKin, "next of kin")
}

// SaveNextOfKin inserts (zero ID) or updates a next of kin, keeping a single primary per crew.
func (r *CrewRecordRepository) SaveNextOfKin(ctx context.Context, k model.NextOfKin) (model.NextOfKin, error) {
	var out model.NextOfKin
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		if err := clearPrimary(ctx, tx, "crew_next_of_kins", k.CrewID, k.ID, k.IsPrimary); err != nil {
			return err
		}
		var err error
		if k.ID == uuid.Nil {
			out, err = scanKin(tx.QueryRow(ctx, `INSERT INTO crew_next_of_kins (crew_id, name, relationship, phone, address, is_primary)
				VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+kinColumns, k.CrewID, k.Name, k.Relationship, k.Phone, k.Address, k.IsPrimary))
		} else {
			out, err = scanKin(tx.QueryRow(ctx, `UPDATE crew_next_of_kins SET name = $3, relationship = $4, phone = $5, address = $6,
				is_primary = $7 WHERE id = $1 AND crew_id = $2 RETURNING `+kinColumns, k.ID, k.CrewID, k.Name, k.Relationship, k.Phone, k.Address, k.IsPrimary))
		}
		if err != nil {
			return fmt.Errorf("save next of kin: %w", err)
		}
		return nil
	})
	return out, err
}

// DeleteNextOfKin removes a next of kin.
func (r *CrewRecordRepository) DeleteNextOfKin(ctx context.Context, crewID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM crew_next_of_kins WHERE id = $1 AND crew_id = $2`, id, crewID)
	return affectedOne(tag, err, "delete next of kin")
}

func clearPrimary(ctx context.Context, tx pgx.Tx, table string, crewID, keepID uuid.UUID, settingPrimary bool) error {
	if !settingPrimary {
		return nil
	}
	if _, err := tx.Exec(ctx, `UPDATE `+table+` SET is_primary = false WHERE crew_id = $1 AND id <> $2 AND is_primary`, crewID, keepID); err != nil {
		return fmt.Errorf("clear primary %s: %w", table, err)
	}
	return nil
}

// DocumentKind selects the travel_documents or certificates table.
type DocumentKind struct {
	Table      string
	TypeTable  string
	TypeColumn string
}

// Document kinds.
var (
	TravelDocuments = DocumentKind{Table: "travel_documents", TypeTable: "document_types", TypeColumn: "document_type_id"}
	Certificates    = DocumentKind{Table: "certificates", TypeTable: "certificate_types", TypeColumn: "certificate_type_id"}
)

func (k DocumentKind) authorityColumn() string {
	if k.Table == "certificates" {
		return "d.issuing_authority"
	}
	return "''"
}

func (k DocumentKind) selectSQL() string {
	return `SELECT d.id, d.crew_id, d.` + k.TypeColumn + `, t.name, t.has_expiry, d.number, d.issued_date, d.expired_date, ` +
		k.authorityColumn() + `, d.place_of_issue, d.file_id FROM ` + k.Table + ` d JOIN ` + k.TypeTable + ` t ON t.id = d.` + k.TypeColumn
}

func scanDocument(row pgx.Row) (model.CrewDocument, error) {
	var d model.CrewDocument
	err := row.Scan(&d.ID, &d.CrewID, &d.TypeID, &d.TypeName, &d.HasExpiry, &d.Number, &d.IssuedDate, &d.ExpiredDate,
		&d.IssuingAuthority, &d.PlaceOfIssue, &d.FileID)
	return d, notFound(err)
}

// ListDocuments returns the documents or certificates of a crew member ordered by type.
func (r *CrewRecordRepository) ListDocuments(ctx context.Context, k DocumentKind, crewID uuid.UUID) ([]model.CrewDocument, error) {
	rows, err := r.pool.Query(ctx, k.selectSQL()+` WHERE d.crew_id = $1 ORDER BY t.sort_order, t.name, d.expired_date DESC NULLS LAST`, crewID)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", k.Table, err)
	}
	return collect(rows, scanDocument, k.Table)
}

// GetDocument returns one document or certificate of a crew member.
func (r *CrewRecordRepository) GetDocument(ctx context.Context, k DocumentKind, crewID, id uuid.UUID) (model.CrewDocument, error) {
	d, err := scanDocument(r.pool.QueryRow(ctx, k.selectSQL()+` WHERE d.id = $1 AND d.crew_id = $2`, id, crewID))
	if err != nil {
		return d, fmt.Errorf("get %s: %w", k.Table, err)
	}
	return d, nil
}

// SaveDocument inserts (zero ID) or updates a document or certificate and returns its ID.
func (r *CrewRecordRepository) SaveDocument(ctx context.Context, k DocumentKind, d model.CrewDocument) (uuid.UUID, error) {
	var id uuid.UUID
	var err error
	if k.Table == "certificates" {
		if d.ID == uuid.Nil {
			err = r.pool.QueryRow(ctx, `INSERT INTO certificates (crew_id, certificate_type_id, number, issued_date, expired_date,
				issuing_authority, place_of_issue, file_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
				d.CrewID, d.TypeID, d.Number, d.IssuedDate, d.ExpiredDate, d.IssuingAuthority, d.PlaceOfIssue, d.FileID).Scan(&id)
		} else {
			err = r.pool.QueryRow(ctx, `UPDATE certificates SET certificate_type_id = $3, number = $4, issued_date = $5, expired_date = $6,
				issuing_authority = $7, place_of_issue = $8, file_id = $9 WHERE id = $1 AND crew_id = $2 RETURNING id`,
				d.ID, d.CrewID, d.TypeID, d.Number, d.IssuedDate, d.ExpiredDate, d.IssuingAuthority, d.PlaceOfIssue, d.FileID).Scan(&id)
		}
	} else {
		if d.ID == uuid.Nil {
			err = r.pool.QueryRow(ctx, `INSERT INTO travel_documents (crew_id, document_type_id, number, issued_date, expired_date,
				place_of_issue, file_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
				d.CrewID, d.TypeID, d.Number, d.IssuedDate, d.ExpiredDate, d.PlaceOfIssue, d.FileID).Scan(&id)
		} else {
			err = r.pool.QueryRow(ctx, `UPDATE travel_documents SET document_type_id = $3, number = $4, issued_date = $5, expired_date = $6,
				place_of_issue = $7, file_id = $8 WHERE id = $1 AND crew_id = $2 RETURNING id`,
				d.ID, d.CrewID, d.TypeID, d.Number, d.IssuedDate, d.ExpiredDate, d.PlaceOfIssue, d.FileID).Scan(&id)
		}
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("save %s: %w", k.Table, notFound(err))
	}
	return id, nil
}

// DeleteDocument removes a document or certificate.
func (r *CrewRecordRepository) DeleteDocument(ctx context.Context, k DocumentKind, crewID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM `+k.Table+` WHERE id = $1 AND crew_id = $2`, id, crewID)
	return affectedOne(tag, err, "delete "+k.Table)
}

// TypeHasExpiry reports whether a document or certificate type has an expiry and is active.
func (r *CrewRecordRepository) TypeHasExpiry(ctx context.Context, k DocumentKind, typeID uuid.UUID) (bool, error) {
	var has bool
	err := r.pool.QueryRow(ctx, `SELECT has_expiry FROM `+k.TypeTable+` WHERE id = $1 AND deleted_at IS NULL`, typeID).Scan(&has)
	if err != nil {
		return false, fmt.Errorf("load %s: %w", k.TypeTable, notFound(err))
	}
	return has, nil
}

const experienceSelect = `SELECT e.id, e.crew_id, e.vessel_name, e.vessel_id, e.rank_id, r.name, e.vessel_type_id, coalesce(vt.name, ''),
	e.grt_hp, e.company_id, coalesce(co.name, e.company_name), e.principal_id, coalesce(pr.name, e.principal_name),
	e.sign_on_date, e.sign_off_date, e.status, e.remarks, e.source, e.contract_id
FROM experiences e
JOIN ranks r ON r.id = e.rank_id
LEFT JOIN vessel_types vt ON vt.id = e.vessel_type_id
LEFT JOIN companies co ON co.id = e.company_id
LEFT JOIN companies pr ON pr.id = e.principal_id`

func scanExperience(row pgx.Row) (model.Experience, error) {
	var x model.Experience
	err := row.Scan(&x.ID, &x.CrewID, &x.VesselName, &x.VesselID, &x.RankID, &x.RankName, &x.VesselTypeID, &x.VesselType,
		&x.GRTHP, &x.CompanyID, &x.CompanyName, &x.PrincipalID, &x.PrincipalName, &x.SignOnDate, &x.SignOffDate, &x.Status,
		&x.Remarks, &x.Source, &x.ContractID)
	return x, notFound(err)
}

// ListExperiences returns the sea service rows of a crew member, newest first.
func (r *CrewRecordRepository) ListExperiences(ctx context.Context, crewID uuid.UUID) ([]model.Experience, error) {
	rows, err := r.pool.Query(ctx, experienceSelect+` WHERE e.crew_id = $1 ORDER BY e.sign_on_date DESC, e.created_at DESC`, crewID)
	if err != nil {
		return nil, fmt.Errorf("list experiences: %w", err)
	}
	return collect(rows, scanExperience, "experiences")
}

// GetExperience returns one sea service row.
func (r *CrewRecordRepository) GetExperience(ctx context.Context, crewID, id uuid.UUID) (model.Experience, error) {
	x, err := scanExperience(r.pool.QueryRow(ctx, experienceSelect+` WHERE e.id = $1 AND e.crew_id = $2`, id, crewID))
	if err != nil {
		return x, fmt.Errorf("get experience: %w", err)
	}
	return x, nil
}

// SaveExperience inserts (zero ID) or updates a manual sea service row and returns its ID.
func (r *CrewRecordRepository) SaveExperience(ctx context.Context, x model.Experience) (uuid.UUID, error) {
	var id uuid.UUID
	var err error
	if x.ID == uuid.Nil {
		err = r.pool.QueryRow(ctx, `INSERT INTO experiences (crew_id, vessel_name, vessel_id, rank_id, vessel_type_id, grt_hp, company_id,
			company_name, principal_id, principal_name, sign_on_date, sign_off_date, status, remarks, source)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, 'manual') RETURNING id`,
			x.CrewID, x.VesselName, x.VesselID, x.RankID, x.VesselTypeID, x.GRTHP, x.CompanyID, x.CompanyName, x.PrincipalID,
			x.PrincipalName, x.SignOnDate, x.SignOffDate, x.Status, x.Remarks).Scan(&id)
	} else {
		err = r.pool.QueryRow(ctx, `UPDATE experiences SET vessel_name = $3, vessel_id = $4, rank_id = $5, vessel_type_id = $6, grt_hp = $7,
			company_id = $8, company_name = $9, principal_id = $10, principal_name = $11, sign_on_date = $12, sign_off_date = $13,
			status = $14, remarks = $15 WHERE id = $1 AND crew_id = $2 AND source = 'manual' RETURNING id`,
			x.ID, x.CrewID, x.VesselName, x.VesselID, x.RankID, x.VesselTypeID, x.GRTHP, x.CompanyID, x.CompanyName, x.PrincipalID,
			x.PrincipalName, x.SignOnDate, x.SignOffDate, x.Status, x.Remarks).Scan(&id)
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("save experience: %w", notFound(err))
	}
	return id, nil
}

// DeleteExperience removes a manual sea service row.
func (r *CrewRecordRepository) DeleteExperience(ctx context.Context, crewID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM experiences WHERE id = $1 AND crew_id = $2 AND source = 'manual'`, id, crewID)
	return affectedOne(tag, err, "delete experience")
}
