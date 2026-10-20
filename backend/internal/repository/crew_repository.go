package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// CrewRepository stores crew records.
type CrewRepository struct {
	pool *pgxpool.Pool
}

// NewCrewRepository returns a CrewRepository.
func NewCrewRepository(pool *pgxpool.Pool) *CrewRepository {
	return &CrewRepository{pool: pool}
}

const crewSelect = `SELECT c.id, c.public_id, c.full_name, c.place_of_birth, c.date_of_birth, c.gender, c.nationality, c.religion,
	c.marital_status, c.height_cm, c.weight_kg, c.shoe_size, c.coverall_size, c.applied_rank_id, c.crew_status_id, c.status_reason,
	c.photo_file_id, c.cv_file_id, c.address, c.city, c.province, c.country, c.postal_code, c.home_phone, c.mobile_phone,
	c.email, c.nearest_airport, c.created_at, c.updated_at,
	coalesce(r.name, ''), coalesce(s.code, ''), s.name, s.color
FROM crews c
JOIN crew_statuses s ON s.id = c.crew_status_id
LEFT JOIN ranks r ON r.id = c.applied_rank_id`

func scanCrew(row pgx.Row) (model.Crew, error) {
	var c model.Crew
	err := row.Scan(&c.ID, &c.PublicID, &c.FullName, &c.PlaceOfBirth, &c.DateOfBirth, &c.Gender, &c.Nationality, &c.Religion,
		&c.MaritalStatus, &c.HeightCM, &c.WeightKG, &c.ShoeSize, &c.CoverallSize, &c.AppliedRankID, &c.CrewStatusID, &c.StatusReason,
		&c.PhotoFileID, &c.CVFileID, &c.Address, &c.City, &c.Province, &c.Country, &c.PostalCode, &c.HomePhone, &c.MobilePhone,
		&c.Email, &c.NearestAirport, &c.CreatedAt, &c.UpdatedAt,
		&c.AppliedRankName, &c.StatusCode, &c.StatusName, &c.StatusColor)
	return c, notFound(err)
}

// Create inserts a crew member with the given public ID and status code.
func (r *CrewRepository) Create(ctx context.Context, c model.Crew, publicID, statusCode string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `INSERT INTO crews (public_id, full_name, place_of_birth, date_of_birth, gender, nationality, religion,
		marital_status, height_cm, weight_kg, shoe_size, coverall_size, applied_rank_id, crew_status_id, photo_file_id, cv_file_id,
		address, city, province, country, postal_code, home_phone, mobile_phone, email, nearest_airport)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, s.id, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25
		FROM crew_statuses s WHERE s.code = $14
		RETURNING id`,
		publicID, c.FullName, c.PlaceOfBirth, c.DateOfBirth, c.Gender, c.Nationality, c.Religion, c.MaritalStatus, c.HeightCM,
		c.WeightKG, c.ShoeSize, c.CoverallSize, c.AppliedRankID, statusCode, c.PhotoFileID, c.CVFileID, c.Address, c.City,
		c.Province, c.Country, c.PostalCode, c.HomePhone, c.MobilePhone, c.Email, c.NearestAirport).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert crew: %w", notFound(err))
	}
	return id, nil
}

// Update replaces the profile fields of a crew member.
func (r *CrewRepository) Update(ctx context.Context, id uuid.UUID, c model.Crew) error {
	tag, err := r.pool.Exec(ctx, `UPDATE crews SET full_name = $2, place_of_birth = $3, date_of_birth = $4, gender = $5,
		nationality = $6, religion = $7, marital_status = $8, height_cm = $9, weight_kg = $10, shoe_size = $11, coverall_size = $12,
		applied_rank_id = $13, photo_file_id = $14, cv_file_id = $15, address = $16, city = $17, province = $18, country = $19,
		postal_code = $20, home_phone = $21, mobile_phone = $22, email = $23, nearest_airport = $24
		WHERE id = $1 AND deleted_at IS NULL`,
		id, c.FullName, c.PlaceOfBirth, c.DateOfBirth, c.Gender, c.Nationality, c.Religion, c.MaritalStatus, c.HeightCM, c.WeightKG,
		c.ShoeSize, c.CoverallSize, c.AppliedRankID, c.PhotoFileID, c.CVFileID, c.Address, c.City, c.Province, c.Country,
		c.PostalCode, c.HomePhone, c.MobilePhone, c.Email, c.NearestAirport)
	if err != nil {
		return fmt.Errorf("update crew: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetByPublicID returns a crew member by public ID.
func (r *CrewRepository) GetByPublicID(ctx context.Context, publicID string) (model.Crew, error) {
	c, err := scanCrew(r.pool.QueryRow(ctx, crewSelect+` WHERE c.public_id = $1 AND c.deleted_at IS NULL`, publicID))
	if err != nil {
		return c, fmt.Errorf("get crew: %w", err)
	}
	return c, nil
}

// GetByID returns a crew member by internal ID.
func (r *CrewRepository) GetByID(ctx context.Context, id uuid.UUID) (model.Crew, error) {
	c, err := scanCrew(r.pool.QueryRow(ctx, crewSelect+` WHERE c.id = $1 AND c.deleted_at IS NULL`, id))
	if err != nil {
		return c, fmt.Errorf("get crew: %w", err)
	}
	return c, nil
}

// SoftDelete marks a crew member deleted.
func (r *CrewRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE crews SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("delete crew: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetStatus moves a crew member to the status with code.
func (r *CrewRepository) SetStatus(ctx context.Context, id uuid.UUID, code, reason string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE crews SET crew_status_id = s.id, status_reason = $3
		FROM crew_statuses s WHERE crews.id = $1 AND s.code = $2 AND crews.deleted_at IS NULL`, id, code, reason)
	if err != nil {
		return fmt.Errorf("set crew status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// HasActiveContract reports whether the crew member holds an Active contract.
func (r *CrewRepository) HasActiveContract(ctx context.Context, crewID uuid.UUID) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM contracts WHERE crew_id = $1 AND status = 'active' AND deleted_at IS NULL)`, crewID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("check active contract: %w", err)
	}
	return ok, nil
}

// ActiveContractRank returns the rank of the crew member's Active contract, if any.
func (r *CrewRepository) ActiveContractRank(ctx context.Context, crewID uuid.UUID) (*uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT rank_id FROM contracts WHERE crew_id = $1 AND status = 'active' AND deleted_at IS NULL`, crewID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("active contract rank: %w", err)
	}
	return &id, nil
}

// Requirements returns the compliance requirements of a rank.
func (r *CrewRepository) Requirements(ctx context.Context, rankID uuid.UUID) ([]RequirementRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT 'document', d.id, d.name, d.has_expiry, d.sort_order FROM compliance_requirements c
		JOIN document_types d ON d.id = c.document_type_id AND d.deleted_at IS NULL
		WHERE c.rank_id = $1
		UNION ALL
		SELECT 'certificate', t.id, t.name, t.has_expiry, 100000 + t.sort_order FROM compliance_requirements c
		JOIN certificate_types t ON t.id = c.certificate_type_id AND t.deleted_at IS NULL
		WHERE c.rank_id = $1
		ORDER BY 5, 3`, rankID)
	if err != nil {
		return nil, fmt.Errorf("list requirements: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (RequirementRow, error) {
		var q RequirementRow
		var sort int
		err := row.Scan(&q.Kind, &q.TypeID, &q.TypeName, &q.HasExpiry, &sort)
		return q, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan requirements: %w", err)
	}
	return out, nil
}

// RequirementRow is one compliance requirement with its type details.
type RequirementRow struct {
	Kind      string
	TypeID    uuid.UUID
	TypeName  string
	HasExpiry bool
}

// HeldRow is one document or certificate held by a crew member.
type HeldRow struct {
	Kind        string
	TypeID      uuid.UUID
	Number      string
	ExpiredDate *time.Time
}

// HeldItems returns all documents and certificates of a crew member.
func (r *CrewRepository) HeldItems(ctx context.Context, crewID uuid.UUID) ([]HeldRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT 'document', document_type_id, number, expired_date FROM travel_documents WHERE crew_id = $1
		UNION ALL
		SELECT 'certificate', certificate_type_id, number, expired_date FROM certificates WHERE crew_id = $1`, crewID)
	if err != nil {
		return nil, fmt.Errorf("list held items: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (HeldRow, error) {
		var h HeldRow
		err := row.Scan(&h.Kind, &h.TypeID, &h.Number, &h.ExpiredDate)
		return h, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan held items: %w", err)
	}
	return out, nil
}
