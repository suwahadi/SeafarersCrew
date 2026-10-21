package job

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Locker grants one caller per key for ttl, so replicas do not repeat a scheduled task.
type Locker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

// Task is a function run on a fixed interval.
type Task struct {
	Name  string
	Every time.Duration
	Slot  func(now time.Time) string
	Run   func(ctx context.Context) error
}

// Scheduler runs tasks periodically inside the API process.
type Scheduler struct {
	tasks  []Task
	locker Locker
	logger *slog.Logger
	now    func() time.Time
}

// NewScheduler builds a Scheduler.
func NewScheduler(tasks []Task, locker Locker, logger *slog.Logger, now func() time.Time) *Scheduler {
	return &Scheduler{tasks: tasks, locker: locker, logger: logger, now: now}
}

// Run starts every task and blocks until ctx is cancelled and the running tasks have finished.
func (s *Scheduler) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, t := range s.tasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.loop(ctx, t)
		}()
	}
	wg.Wait()
}

func (s *Scheduler) loop(ctx context.Context, t Task) {
	s.tick(ctx, t)
	ticker := time.NewTicker(t.Every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx, t)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context, t Task) {
	if ctx.Err() != nil {
		return
	}
	acquired, err := s.locker.Acquire(ctx, t.lockKey(s.now()), t.lockTTL())
	if err != nil {
		s.logger.WarnContext(ctx, "scheduled task lock failed", "task", t.Name, "error", err)
		return
	}
	if !acquired {
		return
	}
	if err := runTask(ctx, t); err != nil && ctx.Err() == nil {
		s.logger.ErrorContext(ctx, "scheduled task failed", "task", t.Name, "error", err)
	}
}

func runTask(ctx context.Context, t Task) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return t.Run(ctx)
}

func (t Task) lockKey(now time.Time) string {
	if t.Slot != nil {
		return "sched:" + t.Name + ":" + t.Slot(now)
	}
	return fmt.Sprintf("sched:%s:%d", t.Name, now.UnixNano()/t.Every.Nanoseconds())
}

func (t Task) lockTTL() time.Duration {
	if t.Slot != nil {
		return 48 * time.Hour
	}
	return 2 * t.Every
}
