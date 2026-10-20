package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// DocumentSourceRepository reads the data merged into generated documents.
type DocumentSourceRepository struct {
	pool *pgxpool.Pool
}

// NewDocumentSourceRepository returns a DocumentSourceRepository.
func NewDocumentSourceRepository(pool *pgxpool.Pool) *DocumentSourceRepository {
	return &DocumentSourceRepository{pool: pool}
}

// CrewProfile is the crew data shared by SEA and CV documents.
type CrewProfile struct {
	ID             uuid.UUID
	PublicID       string
	FullName       string
	DateOfBirth    *time.Time
	PlaceOfBirth   string
	Nationality    string
	Gender         string
	MaritalStatus  string
	Religion       string
	Address        string
	City           string
	Province       string
	Country        string
	PostalCode     string
	MobilePhone    string
	Email          string
	NearestAirport string
	AppliedRank    string
	PhotoFileID    *uuid.UUID
	PassportNo     string
	PassportExpiry *time.Time
	SeamanBookNo   string
}

// AllowanceRow is one contract allowance.
type AllowanceRow struct {
	Name          string
	Basis         string
	Value         decimal.Decimal
	MonthlyAmount decimal.Decimal
}

// SEARecord is the contract data merged into a Seafarer Employment Agreement.
type SEARecord struct {
	ContractID       uuid.UUID
	ContractPublicID string
	Crew             CrewProfile
	VesselName       string
	IMONumber        string
	Flag             string
	Rank             string
	PrincipalName    string
	EmployerName     string
	SignOnDate       time.Time
	PlannedSignOff   time.Time
	DurationMonths   int16
	PointOfHire      string
	SignOnPort       string
	SignOffPort      string
	Currency         string
	CurrencyDecimal  int32
	BasicSalary      decimal.Decimal
	WorkingHours     *int16
	LeaveDays        *decimal.Decimal
	ProbationMonths  *int16
	Allowances       []AllowanceRow
}

const crewProfileColumns = `cr.id, cr.public_id, cr.full_name, cr.date_of_birth, cr.place_of_birth, cr.nationality,
	coalesce(cr.gender, ''), coalesce(cr.marital_status, ''), cr.religion, cr.address, cr.city, cr.province, cr.country,
	cr.postal_code, cr.mobile_phone, cr.email, cr.nearest_airport, coalesce(ar.name, ''), cr.photo_file_id,
	coalesce(pp.number, ''), pp.expired_date, coalesce(sb.number, '')`

const crewProfileJoins = `
LEFT JOIN ranks ar ON ar.id = cr.applied_rank_id
LEFT JOIN LATERAL (SELECT td.number, td.expired_date FROM travel_documents td JOIN document_types dt ON dt.id = td.document_type_id
	WHERE td.crew_id = cr.id AND dt.code = 'passport' ORDER BY td.expired_date DESC NULLS LAST LIMIT 1) pp ON true
LEFT JOIN LATERAL (SELECT td.number FROM travel_documents td JOIN document_types dt ON dt.id = td.document_type_id
	WHERE td.crew_id = cr.id AND dt.code IN ('seaman_book', 'buku_pelaut') ORDER BY td.expired_date DESC NULLS LAST LIMIT 1) sb ON true`

func crewProfileDest(p *CrewProfile) []any {
	return []any{&p.ID, &p.PublicID, &p.FullName, &p.DateOfBirth, &p.PlaceOfBirth, &p.Nationality, &p.Gender, &p.MaritalStatus,
		&p.Religion, &p.Address, &p.City, &p.Province, &p.Country, &p.PostalCode, &p.MobilePhone, &p.Email, &p.NearestAirport,
		&p.AppliedRank, &p.PhotoFileID, &p.PassportNo, &p.PassportExpiry, &p.SeamanBookNo}
}

// CrewProfile returns the crew profile by Crew ID.
func (r *DocumentSourceRepository) CrewProfile(ctx context.Context, publicID string) (CrewProfile, error) {
	var p CrewProfile
	err := r.pool.QueryRow(ctx, `SELECT `+crewProfileColumns+` FROM crews cr`+crewProfileJoins+
		` WHERE cr.public_id = $1 AND cr.deleted_at IS NULL`, publicID).Scan(crewProfileDest(&p)...)
	if err != nil {
		return p, fmt.Errorf("load crew profile: %w", notFound(err))
	}
	return p, nil
}

// SEA returns the contract data for a Seafarer Employment Agreement.
func (r *DocumentSourceRepository) SEA(ctx context.Context, contractPublicID string) (SEARecord, error) {
	var s SEARecord
	var decimals int16
	dest := append([]any{&s.ContractID, &s.ContractPublicID}, crewProfileDest(&s.Crew)...)
	dest = append(dest, &s.VesselName, &s.IMONumber, &s.Flag, &s.Rank, &s.PrincipalName, &s.EmployerName, &s.SignOnDate,
		&s.PlannedSignOff, &s.DurationMonths, &s.PointOfHire, &s.SignOnPort, &s.SignOffPort, &s.Currency, &decimals,
		&s.BasicSalary, &s.WorkingHours, &s.LeaveDays, &s.ProbationMonths)
	err := r.pool.QueryRow(ctx, `SELECT c.id, c.public_id, `+crewProfileColumns+`,
		v.name, v.imo_number, v.flag, rk.name, p.name, e.name, c.sign_on_date, c.planned_sign_off_date, c.duration_months,
		c.point_of_hire, c.sign_on_port, c.sign_off_port, c.currency_code, cur.decimals, c.basic_salary, c.working_hours_week,
		c.leave_days_month, c.probation_months
		FROM contracts c
		JOIN crews cr ON cr.id = c.crew_id
		JOIN vessels v ON v.id = c.vessel_id
		JOIN ranks rk ON rk.id = c.rank_id
		JOIN companies p ON p.id = c.principal_id
		JOIN companies e ON e.id = c.employer_company_id
		JOIN currencies cur ON cur.code = c.currency_code`+crewProfileJoins+`
		WHERE c.public_id = $1 AND c.deleted_at IS NULL`, contractPublicID).Scan(dest...)
	if err != nil {
		return s, fmt.Errorf("load contract for agreement: %w", notFound(err))
	}
	s.CurrencyDecimal = int32(decimals)
	rows, err := r.pool.Query(ctx, `SELECT t.name, a.basis, a.value, a.monthly_amount FROM contract_allowances a
		JOIN allowance_types t ON t.id = a.allowance_type_id WHERE a.contract_id = $1 ORDER BY a.sort_order, t.name`, s.ContractID)
	if err != nil {
		return s, fmt.Errorf("load contract allowances: %w", err)
	}
	s.Allowances, err = collect(rows, func(row pgx.Row) (AllowanceRow, error) {
		var a AllowanceRow
		err := row.Scan(&a.Name, &a.Basis, &a.Value, &a.MonthlyAmount)
		return a, err
	}, "contract allowances")
	return s, err
}

// ContractID resolves a contract public ID.
func (r *DocumentSourceRepository) ContractID(ctx context.Context, publicID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM contracts WHERE public_id = $1 AND deleted_at IS NULL`, publicID).Scan(&id)
	if err != nil {
		return id, fmt.Errorf("resolve contract: %w", notFound(err))
	}
	return id, nil
}
