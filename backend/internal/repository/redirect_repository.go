package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

var slugTables = map[string]string{
	"pages":     `SELECT EXISTS (SELECT 1 FROM pages WHERE slug = $1 AND id <> $2 AND deleted_at IS NULL)`,
	"news":      `SELECT EXISTS (SELECT 1 FROM news WHERE slug = $1 AND id <> $2 AND deleted_at IS NULL)`,
	"vacancies": `SELECT EXISTS (SELECT 1 FROM vacancies WHERE slug = $1 AND id <> $2 AND deleted_at IS NULL)`,
}

// SlugTaken reports whether another live row of table already uses slug.
func SlugTaken(ctx context.Context, db DBTX, table, slug string, exclude uuid.UUID) (bool, error) {
	query, ok := slugTables[table]
	if !ok {
		return false, fmt.Errorf("slug check: unknown table %q", table)
	}
	var taken bool
	if err := db.QueryRow(ctx, query, slug, exclude).Scan(&taken); err != nil {
		return false, fmt.Errorf("slug check on %s: %w", table, err)
	}
	return taken, nil
}

// SaveRedirect stores a permanent redirect from one path to another, pointing earlier
// redirects at the new target and dropping any redirect away from a path that is live again.
func SaveRedirect(ctx context.Context, db DBTX, from, to string) error {
	steps := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM redirects WHERE from_path = $1`, []any{to}},
		{`UPDATE redirects SET to_path = $2 WHERE to_path = $1`, []any{from, to}},
		{`INSERT INTO redirects (from_path, to_path, status_code) VALUES ($1, $2, 301)
			ON CONFLICT (from_path) DO UPDATE SET to_path = EXCLUDED.to_path, status_code = 301`, []any{from, to}},
		{`DELETE FROM redirects WHERE from_path = to_path`, nil},
	}
	for _, s := range steps {
		if _, err := db.Exec(ctx, s.sql, s.args...); err != nil {
			return fmt.Errorf("save redirect %s: %w", from, err)
		}
	}
	return nil
}

// DeleteRedirectFrom removes the redirect away from path, used when path becomes live again.
func DeleteRedirectFrom(ctx context.Context, db DBTX, path string) error {
	if _, err := db.Exec(ctx, `DELETE FROM redirects WHERE from_path = $1`, path); err != nil {
		return fmt.Errorf("delete redirect %s: %w", path, err)
	}
	return nil
}

// RedirectRepository reads redirects for the public API.
type RedirectRepository struct {
	pool *pgxpool.Pool
}

// NewRedirectRepository builds a RedirectRepository.
func NewRedirectRepository(pool *pgxpool.Pool) *RedirectRepository {
	return &RedirectRepository{pool: pool}
}

// Resolve returns the redirect away from path.
func (r *RedirectRepository) Resolve(ctx context.Context, path string) (model.Redirect, error) {
	var rd model.Redirect
	err := r.pool.QueryRow(ctx, `SELECT from_path, to_path, status_code FROM redirects WHERE from_path = $1`, path).
		Scan(&rd.FromPath, &rd.ToPath, &rd.StatusCode)
	if err != nil {
		return rd, fmt.Errorf("resolve redirect: %w", notFound(err))
	}
	return rd, nil
}
