// Package job runs background jobs from a Redis queue and bounds CPU-heavy work.
package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

const (
	readyKey   = "jobs:ready"
	delayedKey = "jobs:delayed"
)

// Store persists job state.
type Store interface {
	Create(ctx context.Context, kind string, payload json.RawMessage) (model.Job, error)
	Get(ctx context.Context, id uuid.UUID) (model.Job, error)
	Start(ctx context.Context, id uuid.UUID) (model.Job, error)
	Succeed(ctx context.Context, id uuid.UUID, result json.RawMessage) error
	Retry(ctx context.Context, id uuid.UUID, msg string) error
	Fail(ctx context.Context, id uuid.UUID, msg string) error
	Stale(ctx context.Context, olderThan string) ([]uuid.UUID, error)
}

// Handler executes one job and returns a JSON-serializable result.
type Handler func(ctx context.Context, j model.Job) (any, error)

// Permanent wraps an error that must not be retried.
type Permanent struct{ Err error }

func (p Permanent) Error() string { return p.Err.Error() }

func (p Permanent) Unwrap() error { return p.Err }

// Queue enqueues jobs and runs workers.
type Queue struct {
	store       Store
	rdb         *goredis.Client
	logger      *slog.Logger
	mu          sync.RWMutex
	handlers    map[string]Handler
	maxAttempts int
	timeout     time.Duration
}

// NewQueue returns a Queue.
func NewQueue(store Store, rdb *goredis.Client, logger *slog.Logger) *Queue {
	return &Queue{store: store, rdb: rdb, logger: logger, handlers: map[string]Handler{}, maxAttempts: 3, timeout: 15 * time.Minute}
}

// Register binds a handler to a job kind.
func (q *Queue) Register(kind string, h Handler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[kind] = h
}

// Enqueue stores and queues a job.
func (q *Queue) Enqueue(ctx context.Context, kind string, payload any) (model.Job, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return model.Job{}, fmt.Errorf("encode job payload: %w", err)
	}
	j, err := q.store.Create(ctx, kind, raw)
	if err != nil {
		return model.Job{}, err
	}
	if err := q.rdb.LPush(ctx, readyKey, j.ID.String()).Err(); err != nil {
		return model.Job{}, fmt.Errorf("enqueue job: %w", err)
	}
	return j, nil
}

// Get returns a job by ID.
func (q *Queue) Get(ctx context.Context, id uuid.UUID) (model.Job, error) {
	return q.store.Get(ctx, id)
}

// Run starts workers and the delayed-job mover until ctx is cancelled.
func (q *Queue) Run(ctx context.Context, workers int) {
	var wg sync.WaitGroup
	wg.Add(workers + 1)
	go func() {
		defer wg.Done()
		q.promoteLoop(ctx)
	}()
	for range workers {
		go func() {
			defer wg.Done()
			q.workLoop(ctx)
		}()
	}
	q.requeueStale(ctx)
	wg.Wait()
}

func (q *Queue) workLoop(ctx context.Context) {
	for ctx.Err() == nil {
		res, err := q.rdb.BRPop(ctx, 5*time.Second, readyKey).Result()
		if errors.Is(err, goredis.Nil) || ctx.Err() != nil {
			continue
		}
		if err != nil {
			q.logger.ErrorContext(ctx, "job queue pop failed", "error", err)
			sleep(ctx, time.Second)
			continue
		}
		id, err := uuid.Parse(res[1])
		if err != nil {
			continue
		}
		q.process(ctx, id)
	}
}

func (q *Queue) process(ctx context.Context, id uuid.UUID) {
	j, err := q.store.Start(ctx, id)
	if err != nil {
		q.logger.WarnContext(ctx, "job start skipped", "job_id", id, "error", err)
		return
	}
	q.mu.RLock()
	h, ok := q.handlers[j.Kind]
	q.mu.RUnlock()
	if !ok {
		q.finishErr(ctx, j, Permanent{Err: fmt.Errorf("no handler for job kind %q", j.Kind)})
		return
	}
	jctx, cancel := context.WithTimeout(ctx, q.timeout)
	defer cancel()
	result, err := runSafely(jctx, h, j)
	if err != nil {
		q.finishErr(ctx, j, err)
		return
	}
	raw, err := json.Marshal(result)
	if err != nil {
		q.finishErr(ctx, j, Permanent{Err: fmt.Errorf("encode job result: %w", err)})
		return
	}
	if err := q.store.Succeed(ctx, j.ID, raw); err != nil {
		q.logger.ErrorContext(ctx, "job result not saved", "job_id", j.ID, "error", err)
	}
}

func runSafely(ctx context.Context, h Handler, j model.Job) (result any, err error) {
	defer func() {
		if v := recover(); v != nil {
			err = Permanent{Err: fmt.Errorf("job panicked: %v", v)}
		}
	}()
	return h(ctx, j)
}

func (q *Queue) finishErr(ctx context.Context, j model.Job, jobErr error) {
	var perm Permanent
	if errors.As(jobErr, &perm) || j.Attempts >= q.maxAttempts {
		q.logger.WarnContext(ctx, "job failed", "job_id", j.ID, "kind", j.Kind, "attempts", j.Attempts, "error", jobErr)
		if err := q.store.Fail(ctx, j.ID, jobErr.Error()); err != nil {
			q.logger.ErrorContext(ctx, "job failure not saved", "job_id", j.ID, "error", err)
		}
		return
	}
	if err := q.store.Retry(ctx, j.ID, jobErr.Error()); err != nil {
		q.logger.ErrorContext(ctx, "job retry not saved", "job_id", j.ID, "error", err)
		return
	}
	due := time.Now().Add(Backoff(j.Attempts))
	if err := q.rdb.ZAdd(ctx, delayedKey, goredis.Z{Score: float64(due.Unix()), Member: j.ID.String()}).Err(); err != nil {
		q.logger.ErrorContext(ctx, "job retry not scheduled", "job_id", j.ID, "error", err)
	}
}

// Backoff returns the delay before retry number attempt (1-based): 30s, 2m, 8m.
func Backoff(attempt int) time.Duration {
	return time.Duration(30*math.Pow(4, float64(max(attempt-1, 0)))) * time.Second
}

var promoteDue = goredis.NewScript(`
local ids = redis.call("ZRANGEBYSCORE", KEYS[1], "-inf", ARGV[1], "LIMIT", 0, 100)
for _, id in ipairs(ids) do
  redis.call("ZREM", KEYS[1], id)
  redis.call("LPUSH", KEYS[2], id)
end
return #ids
`)

func (q *Queue) promoteLoop(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := promoteDue.Run(ctx, q.rdb, []string{delayedKey, readyKey}, strconv.FormatInt(time.Now().Unix(), 10)).Err(); err != nil && ctx.Err() == nil {
				q.logger.ErrorContext(ctx, "delayed jobs not promoted", "error", err)
			}
		}
	}
}

func (q *Queue) requeueStale(ctx context.Context) {
	ids, err := q.store.Stale(ctx, "30 minutes")
	if err != nil {
		q.logger.ErrorContext(ctx, "stale jobs not listed", "error", err)
		return
	}
	for _, id := range ids {
		if err := q.rdb.LPush(ctx, readyKey, id.String()).Err(); err != nil {
			q.logger.ErrorContext(ctx, "stale job not requeued", "job_id", id, "error", err)
		}
	}
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
