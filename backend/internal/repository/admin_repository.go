package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// AdminRepository stores the administrator and recovery codes.
type AdminRepository struct {
	pool *pgxpool.Pool
}

// NewAdminRepository returns an AdminRepository.
func NewAdminRepository(pool *pgxpool.Pool) *AdminRepository {
	return &AdminRepository{pool: pool}
}

const adminColumns = `id, name, email, password_hash, password_changed_at, must_change_password, created_at, updated_at`

func scanAdmin(row pgx.Row) (model.Admin, error) {
	var a model.Admin
	err := row.Scan(&a.ID, &a.Name, &a.Email, &a.PasswordHash, &a.PasswordChangedAt, &a.MustChangePassword, &a.CreatedAt, &a.UpdatedAt)
	return a, notFound(err)
}

// Count returns the number of admins.
func (r *AdminRepository) Count(ctx context.Context) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM admins`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count admins: %w", err)
	}
	return n, nil
}

// FindByEmail returns the admin with a case-insensitive email match.
func (r *AdminRepository) FindByEmail(ctx context.Context, email string) (model.Admin, error) {
	a, err := scanAdmin(r.pool.QueryRow(ctx, `SELECT `+adminColumns+` FROM admins WHERE lower(email) = lower($1)`, email))
	if err != nil {
		return a, fmt.Errorf("find admin by email: %w", err)
	}
	return a, nil
}

// FindByID returns the admin with the given ID.
func (r *AdminRepository) FindByID(ctx context.Context, id uuid.UUID) (model.Admin, error) {
	a, err := scanAdmin(r.pool.QueryRow(ctx, `SELECT `+adminColumns+` FROM admins WHERE id = $1`, id))
	if err != nil {
		return a, fmt.Errorf("find admin by id: %w", err)
	}
	return a, nil
}

// CreateWithRecoveryCodes inserts the admin and its hashed recovery codes atomically.
func (r *AdminRepository) CreateWithRecoveryCodes(ctx context.Context, a model.Admin, codeHashes []string) (model.Admin, error) {
	var created model.Admin
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		created, err = scanAdmin(tx.QueryRow(ctx,
			`INSERT INTO admins (name, email, password_hash, must_change_password) VALUES ($1, $2, $3, $4) RETURNING `+adminColumns,
			a.Name, a.Email, a.PasswordHash, a.MustChangePassword))
		if err != nil {
			return fmt.Errorf("insert admin: %w", err)
		}
		return insertRecoveryCodes(ctx, tx, created.ID, codeHashes)
	})
	return created, err
}

// UpdatePassword stores a new password hash and must-change flag.
func (r *AdminRepository) UpdatePassword(ctx context.Context, id uuid.UUID, hash string, mustChange bool) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE admins SET password_hash = $2, must_change_password = $3, password_changed_at = now() WHERE id = $1`,
		id, hash, mustChange)
	if err != nil {
		return fmt.Errorf("update admin password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateProfile changes the admin name and email.
func (r *AdminRepository) UpdateProfile(ctx context.Context, id uuid.UUID, name, email string) (model.Admin, error) {
	a, err := scanAdmin(r.pool.QueryRow(ctx,
		`UPDATE admins SET name = $2, email = $3 WHERE id = $1 RETURNING `+adminColumns, id, name, email))
	if err != nil {
		return a, fmt.Errorf("update admin profile: %w", err)
	}
	return a, nil
}

// UnusedRecoveryCodes lists codes that have not been used.
func (r *AdminRepository) UnusedRecoveryCodes(ctx context.Context, adminID uuid.UUID) ([]model.RecoveryCode, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, admin_id, code_hash, used_at FROM recovery_codes WHERE admin_id = $1 AND used_at IS NULL ORDER BY created_at`, adminID)
	if err != nil {
		return nil, fmt.Errorf("list recovery codes: %w", err)
	}
	codes, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.RecoveryCode, error) {
		var c model.RecoveryCode
		err := row.Scan(&c.ID, &c.AdminID, &c.CodeHash, &c.UsedAt)
		return c, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan recovery codes: %w", err)
	}
	return codes, nil
}

// MarkRecoveryCodeUsed consumes a code and reports whether it was still unused.
func (r *AdminRepository) MarkRecoveryCodeUsed(ctx context.Context, id uuid.UUID, at time.Time) (bool, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE recovery_codes SET used_at = $2 WHERE id = $1 AND used_at IS NULL`, id, at)
	if err != nil {
		return false, fmt.Errorf("mark recovery code used: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// ReplaceRecoveryCodes deletes all codes of the admin and stores a new set.
func (r *AdminRepository) ReplaceRecoveryCodes(ctx context.Context, adminID uuid.UUID, codeHashes []string) error {
	return InTx(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM recovery_codes WHERE admin_id = $1`, adminID); err != nil {
			return fmt.Errorf("delete recovery codes: %w", err)
		}
		return insertRecoveryCodes(ctx, tx, adminID, codeHashes)
	})
}

func insertRecoveryCodes(ctx context.Context, tx pgx.Tx, adminID uuid.UUID, hashes []string) error {
	rows := make([][]any, len(hashes))
	for i, h := range hashes {
		rows[i] = []any{adminID, h}
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"recovery_codes"}, []string{"admin_id", "code_hash"}, pgx.CopyFromRows(rows)); err != nil {
		return fmt.Errorf("insert recovery codes: %w", err)
	}
	return nil
}
