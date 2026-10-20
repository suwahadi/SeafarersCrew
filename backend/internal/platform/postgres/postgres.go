// Package postgres opens the pgx pool and runs schema migrations.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/suwahadi/seafarerscrew/backend/database/migrations"
)

// Open creates a connection pool sized from the CPU count and verifies connectivity.
func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	if !strings.Contains(databaseURL, "pool_max_conns") {
		cfg.MaxConns = poolSize(runtime.NumCPU())
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

func poolSize(cpus int) int32 {
	switch {
	case cpus <= 1:
		return 4
	case cpus >= 32:
		return 128
	default:
		return int32(cpus) * 4
	}
}

// Migrate applies (up) or rolls back one step (down) of the embedded migrations.
func Migrate(databaseURL, direction string) error {
	src, err := iofs.New(migrations.Files, ".")
	if err != nil {
		return fmt.Errorf("open migration source: %w", err)
	}
	connCfg, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("parse database url: %w", err)
	}
	db := stdlib.OpenDB(*connCfg)
	defer func() { _ = db.Close() }()
	driver, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{})
	if err != nil {
		return fmt.Errorf("open migration driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	switch direction {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-1)
	default:
		return fmt.Errorf("unknown migration direction %q", direction)
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate %s: %w", direction, err)
	}
	return nil
}

// IsUniqueViolation reports whether err is a unique-constraint violation, optionally on a named constraint.
func IsUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	return constraint == "" || pgErr.ConstraintName == constraint
}

// IsForeignKeyViolation reports whether err is a foreign-key violation.
func IsForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// ForeignKeyColumn returns the column of a violated <table>_<column>_fkey constraint on table.
func ForeignKeyColumn(err error, table string) (string, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(pgErr.ConstraintName, table+"_"), "_fkey")
	return name, name != pgErr.ConstraintName
}

// IsCheckViolation reports whether err is a CHECK constraint violation.
func IsCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}
