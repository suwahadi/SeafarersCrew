package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// JobRepository stores background job state.
type JobRepository struct {
	pool *pgxpool.Pool
}

// NewJobRepository returns a JobRepository.
func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

const jobColumns = `id, kind, status, payload, result, coalesce(error, ''), attempts, started_at, finished_at, created_at`

func scanJob(row pgx.Row) (model.Job, error) {
	var j model.Job
	err := row.Scan(&j.ID, &j.Kind, &j.Status, &j.Payload, &j.Result, &j.Error, &j.Attempts, &j.StartedAt, &j.FinishedAt, &j.CreatedAt)
	return j, notFound(err)
}

// Create inserts a queued job.
func (r *JobRepository) Create(ctx context.Context, kind string, payload json.RawMessage) (model.Job, error) {
	j, err := scanJob(r.pool.QueryRow(ctx, `INSERT INTO jobs (kind, payload) VALUES ($1, $2) RETURNING `+jobColumns, kind, payload))
	if err != nil {
		return j, fmt.Errorf("create job: %w", err)
	}
	return j, nil
}

// Get returns a job by ID.
func (r *JobRepository) Get(ctx context.Context, id uuid.UUID) (model.Job, error) {
	j, err := scanJob(r.pool.QueryRow(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id = $1`, id))
	if err != nil {
		return j, fmt.Errorf("get job: %w", err)
	}
	return j, nil
}

// Start marks a job running and increments its attempts.
func (r *JobRepository) Start(ctx context.Context, id uuid.UUID) (model.Job, error) {
	j, err := scanJob(r.pool.QueryRow(ctx, `UPDATE jobs SET status = 'running', attempts = attempts + 1, started_at = now(), error = NULL
		WHERE id = $1 AND status IN ('queued', 'running') RETURNING `+jobColumns, id))
	if err != nil {
		return j, fmt.Errorf("start job: %w", err)
	}
	return j, nil
}

// Succeed stores the result of a finished job.
func (r *JobRepository) Succeed(ctx context.Context, id uuid.UUID, result json.RawMessage) error {
	if _, err := r.pool.Exec(ctx, `UPDATE jobs SET status = 'succeeded', result = $2, finished_at = now() WHERE id = $1`, id, result); err != nil {
		return fmt.Errorf("finish job: %w", err)
	}
	return nil
}

// Retry records an error and puts the job back in the queued state.
func (r *JobRepository) Retry(ctx context.Context, id uuid.UUID, msg string) error {
	if _, err := r.pool.Exec(ctx, `UPDATE jobs SET status = 'queued', error = $2 WHERE id = $1`, id, msg); err != nil {
		return fmt.Errorf("retry job: %w", err)
	}
	return nil
}

// Fail records a terminal error.
func (r *JobRepository) Fail(ctx context.Context, id uuid.UUID, msg string) error {
	if _, err := r.pool.Exec(ctx, `UPDATE jobs SET status = 'failed', error = $2, finished_at = now() WHERE id = $1`, id, msg); err != nil {
		return fmt.Errorf("fail job: %w", err)
	}
	return nil
}

// Stale returns queued or running jobs untouched for longer than the given interval, for re-enqueueing after a restart.
func (r *JobRepository) Stale(ctx context.Context, olderThan string) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM jobs WHERE status IN ('queued', 'running') AND updated_at < now() - $1::interval ORDER BY created_at LIMIT 500`, olderThan)
	if err != nil {
		return nil, fmt.Errorf("list stale jobs: %w", err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("scan stale jobs: %w", err)
	}
	return ids, nil
}
