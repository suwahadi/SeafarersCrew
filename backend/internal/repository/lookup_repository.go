package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LookupRepository answers existence and type questions used by master data rules.
type LookupRepository struct {
	pool *pgxpool.Pool
}

// NewLookupRepository returns a LookupRepository.
func NewLookupRepository(pool *pgxpool.Pool) *LookupRepository {
	return &LookupRepository{pool: pool}
}

// CompanyType returns the type of a non-deleted company.
func (r *LookupRepository) CompanyType(ctx context.Context, id uuid.UUID) (string, error) {
	var t string
	if err := r.pool.QueryRow(ctx, `SELECT type FROM companies WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&t); err != nil {
		return "", fmt.Errorf("company type: %w", notFound(err))
	}
	return t, nil
}

// VesselTypeExists reports whether a non-deleted vessel type exists.
func (r *LookupRepository) VesselTypeExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.exists(ctx, `SELECT EXISTS (SELECT 1 FROM vessel_types WHERE id = $1 AND deleted_at IS NULL)`, id)
}

// FileExists reports whether a non-deleted stored file exists.
func (r *LookupRepository) FileExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.exists(ctx, `SELECT EXISTS (SELECT 1 FROM files WHERE id = $1 AND deleted_at IS NULL)`, id)
}

func (r *LookupRepository) exists(ctx context.Context, sql string, id uuid.UUID) (bool, error) {
	var ok bool
	if err := r.pool.QueryRow(ctx, sql, id).Scan(&ok); err != nil {
		return false, fmt.Errorf("lookup: %w", err)
	}
	return ok, nil
}
