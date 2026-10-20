package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// SettingRepository stores settings as key-value rows.
type SettingRepository struct {
	pool *pgxpool.Pool
}

// NewSettingRepository returns a SettingRepository.
func NewSettingRepository(pool *pgxpool.Pool) *SettingRepository {
	return &SettingRepository{pool: pool}
}

// ListGroup returns all rows of a group.
func (r *SettingRepository) ListGroup(ctx context.Context, group string) ([]model.Setting, error) {
	rows, err := r.pool.Query(ctx, `SELECT group_name, key, value, is_secret FROM settings WHERE group_name = $1 ORDER BY key`, group)
	if err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	settings, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Setting, error) {
		var s model.Setting
		err := row.Scan(&s.Group, &s.Key, &s.Value, &s.IsSecret)
		return s, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan settings: %w", err)
	}
	return settings, nil
}

// Upsert writes rows, replacing existing values for the same group and key.
func (r *SettingRepository) Upsert(ctx context.Context, settings []model.Setting) error {
	return InTx(ctx, r.pool, func(tx pgx.Tx) error {
		batch := &pgx.Batch{}
		for _, s := range settings {
			batch.Queue(`INSERT INTO settings (group_name, key, value, is_secret) VALUES ($1, $2, $3, $4)
				ON CONFLICT (group_name, key) DO UPDATE SET value = EXCLUDED.value, is_secret = EXCLUDED.is_secret`,
				s.Group, s.Key, s.Value, s.IsSecret)
		}
		if err := tx.SendBatch(ctx, batch).Close(); err != nil {
			return fmt.Errorf("upsert settings: %w", err)
		}
		return nil
	})
}

// InsertMissing writes rows only where the group and key do not exist yet.
func (r *SettingRepository) InsertMissing(ctx context.Context, settings []model.Setting) error {
	batch := &pgx.Batch{}
	for _, s := range settings {
		batch.Queue(`INSERT INTO settings (group_name, key, value, is_secret) VALUES ($1, $2, $3, $4) ON CONFLICT (group_name, key) DO NOTHING`,
			s.Group, s.Key, s.Value, s.IsSecret)
	}
	if err := r.pool.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("insert default settings: %w", err)
	}
	return nil
}
