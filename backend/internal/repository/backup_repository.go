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

// BackupRepository stores backup run history.
type BackupRepository struct {
	pool *pgxpool.Pool
}

// NewBackupRepository builds a BackupRepository.
func NewBackupRepository(pool *pgxpool.Pool) *BackupRepository {
	return &BackupRepository{pool: pool}
}

const backupColumns = `id, coalesce(storage_key, ''), coalesce(size_bytes, 0), status, started_at, finished_at, coalesce(error, '')`

func scanBackup(row pgx.Row) (model.Backup, error) {
	var b model.Backup
	err := row.Scan(&b.ID, &b.StorageKey, &b.SizeBytes, &b.Status, &b.StartedAt, &b.FinishedAt, &b.Error)
	return b, notFound(err)
}

// Create inserts a running backup.
func (r *BackupRepository) Create(ctx context.Context, startedAt time.Time) (model.Backup, error) {
	b, err := scanBackup(r.pool.QueryRow(ctx, `INSERT INTO backups (status, started_at) VALUES ('running', $1) RETURNING `+backupColumns, startedAt))
	if err != nil {
		return b, fmt.Errorf("create backup: %w", err)
	}
	return b, nil
}

// Get returns one backup.
func (r *BackupRepository) Get(ctx context.Context, id uuid.UUID) (model.Backup, error) {
	b, err := scanBackup(r.pool.QueryRow(ctx, `SELECT `+backupColumns+` FROM backups WHERE id = $1`, id))
	if err != nil {
		return b, fmt.Errorf("get backup: %w", err)
	}
	return b, nil
}

// Finish records the outcome of a backup.
func (r *BackupRepository) Finish(ctx context.Context, id uuid.UUID, status, storageKey string, size int64, errText string, at time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE backups SET status = $2, storage_key = nullif($3, ''), size_bytes = nullif($4, 0), error = nullif($5, ''), finished_at = $6 WHERE id = $1`,
		id, status, storageKey, size, errText, at)
	if err != nil {
		return fmt.Errorf("finish backup: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// List returns backups newest first.
func (r *BackupRepository) List(ctx context.Context, page, perPage int) (Page[model.Backup], error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM backups`).Scan(&total); err != nil {
		return Page[model.Backup]{}, fmt.Errorf("count backups: %w", err)
	}
	rows, err := r.pool.Query(ctx, `SELECT `+backupColumns+` FROM backups ORDER BY started_at DESC, id DESC LIMIT $1 OFFSET $2`, perPage, (page-1)*perPage)
	if err != nil {
		return Page[model.Backup]{}, fmt.Errorf("list backups: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Backup, error) { return scanBackup(row) })
	if err != nil {
		return Page[model.Backup]{}, fmt.Errorf("scan backups: %w", err)
	}
	return Page[model.Backup]{Items: items, Total: total}, nil
}

// Since returns the backups started at or after t, newest first.
func (r *BackupRepository) Since(ctx context.Context, t time.Time) ([]model.Backup, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+backupColumns+` FROM backups WHERE started_at >= $1 ORDER BY started_at DESC`, t)
	if err != nil {
		return nil, fmt.Errorf("list recent backups: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Backup, error) { return scanBackup(row) })
	if err != nil {
		return nil, fmt.Errorf("scan recent backups: %w", err)
	}
	return items, nil
}

// FailStale marks backups that have been running since before cutoff as failed.
func (r *BackupRepository) FailStale(ctx context.Context, cutoff, at time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE backups SET status = 'failed', error = 'The backup did not finish.', finished_at = $2 WHERE status = 'running' AND started_at < $1`, cutoff, at)
	if err != nil {
		return 0, fmt.Errorf("fail stale backups: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Expired returns the backups started before cutoff, excluding running ones.
func (r *BackupRepository) Expired(ctx context.Context, cutoff time.Time) ([]model.Backup, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+backupColumns+` FROM backups WHERE started_at < $1 AND status <> 'running' ORDER BY started_at`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("list expired backups: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Backup, error) { return scanBackup(row) })
	if err != nil {
		return nil, fmt.Errorf("scan expired backups: %w", err)
	}
	return items, nil
}

// Delete removes backup rows.
func (r *BackupRepository) Delete(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := r.pool.Exec(ctx, `DELETE FROM backups WHERE id = ANY($1)`, ids); err != nil {
		return fmt.Errorf("delete backups: %w", err)
	}
	return nil
}
