package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// FileRepository records stored files.
type FileRepository struct {
	pool *pgxpool.Pool
}

// NewFileRepository returns a FileRepository.
func NewFileRepository(pool *pgxpool.Pool) *FileRepository {
	return &FileRepository{pool: pool}
}

const fileColumns = `id, storage_key, original_name, mime_type, size_original, size_stored, sha256,
	coalesce(owner_type, ''), owner_id, is_public, variants, coalesce(width, 0), coalesce(height, 0), status, created_at`

func scanFile(row pgx.Row) (model.File, error) {
	var f model.File
	err := row.Scan(&f.ID, &f.StorageKey, &f.OriginalName, &f.MimeType, &f.SizeOriginal, &f.SizeStored, &f.SHA256,
		&f.OwnerType, &f.OwnerID, &f.IsPublic, &f.Variants, &f.Width, &f.Height, &f.Status, &f.CreatedAt)
	return f, notFound(err)
}

// Create inserts a file row.
func (r *FileRepository) Create(ctx context.Context, f model.File) (model.File, error) {
	if f.Variants == nil {
		f.Variants = []model.FileVariant{}
	}
	created, err := scanFile(r.pool.QueryRow(ctx, `INSERT INTO files
		(storage_key, original_name, mime_type, size_original, size_stored, sha256, owner_type, owner_id, is_public, variants, width, height, status)
		VALUES ($1, $2, $3, $4, $5, $6, nullif($7, ''), $8, $9, $10, nullif($11, 0), nullif($12, 0), $13) RETURNING `+fileColumns,
		f.StorageKey, f.OriginalName, f.MimeType, f.SizeOriginal, f.SizeStored, f.SHA256, f.OwnerType, f.OwnerID, f.IsPublic, f.Variants, f.Width, f.Height, f.Status))
	if err != nil {
		return created, fmt.Errorf("create file: %w", err)
	}
	return created, nil
}

// Get returns a non-deleted file.
func (r *FileRepository) Get(ctx context.Context, id uuid.UUID) (model.File, error) {
	f, err := scanFile(r.pool.QueryRow(ctx, `SELECT `+fileColumns+` FROM files WHERE id = $1 AND deleted_at IS NULL`, id))
	if err != nil {
		return f, fmt.Errorf("get file: %w", err)
	}
	return f, nil
}

// SoftDelete marks a file deleted.
func (r *FileRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE files SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListByOwner returns the non-deleted files of an owner, newest first.
func (r *FileRepository) ListByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) ([]model.File, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+fileColumns+` FROM files
		WHERE owner_type = $1 AND owner_id = $2 AND deleted_at IS NULL ORDER BY created_at DESC`, ownerType, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	files, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.File, error) { return scanFile(row) })
	if err != nil {
		return nil, fmt.Errorf("scan files: %w", err)
	}
	return files, nil
}

// AttachOwner links a file to the record that uses it.
func (r *FileRepository) AttachOwner(ctx context.Context, id uuid.UUID, ownerType string, ownerID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE files SET owner_type = $2, owner_id = $3 WHERE id = $1 AND deleted_at IS NULL`, id, ownerType, ownerID)
	if err != nil {
		return fmt.Errorf("attach file owner: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
