package job

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type memLocker struct {
	mu   sync.Mutex
	keys map[string]bool
	ttls map[string]time.Duration
}

func newMemLocker() *memLocker {
	return &memLocker{keys: map[string]bool{}, ttls: map[string]time.Duration{}}
}

func (m *memLocker) Acquire(_ context.Context, key string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.keys[key] {
		return false, nil
	}
	m.keys[key], m.ttls[key] = true, ttl
	return true, nil
}

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestTaskLockKeyChangesEveryInterval(t *testing.T) {
	task := Task{Name: "close", Every: time.Hour}
	base := time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
	assert.Equal(t, task.lockKey(base), task.lockKey(base.Add(59*time.Minute)))
	assert.NotEqual(t, task.lockKey(base), task.lockKey(base.Add(time.Hour)))
	assert.Equal(t, 2*time.Hour, task.lockTTL())
}

func TestTaskWithSlotLocksPerSlot(t *testing.T) {
	task := Task{Name: "purge", Every: 6 * time.Hour, Slot: func(now time.Time) string { return now.Format("2006-01") }}
	assert.Equal(t, "sched:purge:2026-10", task.lockKey(time.Date(2026, 10, 3, 1, 0, 0, 0, time.UTC)))
	assert.Equal(t, task.lockKey(time.Date(2026, 10, 3, 1, 0, 0, 0, time.UTC)), task.lockKey(time.Date(2026, 10, 30, 1, 0, 0, 0, time.UTC)))
	assert.NotEqual(t, task.lockKey(time.Date(2026, 10, 30, 1, 0, 0, 0, time.UTC)), task.lockKey(time.Date(2026, 11, 1, 1, 0, 0, 0, time.UTC)))
}

func TestSchedulerRunsTasksOnStartAndOnEveryTick(t *testing.T) {
	var runs atomic.Int32
	now := time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	clock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		now = now.Add(time.Hour)
		return now
	}
	s := NewScheduler([]Task{{Name: "tick", Every: 20 * time.Millisecond, Run: func(context.Context) error { runs.Add(1); return nil }}}, newMemLocker(), quietLogger(), clock)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	s.Run(ctx)
	assert.GreaterOrEqual(t, runs.Load(), int32(3))
}

func TestSchedulerSkipsATaskAnotherReplicaAlreadyTook(t *testing.T) {
	var runs atomic.Int32
	locker := newMemLocker()
	fixed := func() time.Time { return time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC) }
	task := Task{Name: "once", Every: time.Hour, Run: func(context.Context) error { runs.Add(1); return nil }}
	for range 2 {
		s := NewScheduler([]Task{task}, locker, quietLogger(), fixed)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		s.tick(context.Background(), task)
		s.Run(ctx)
	}
	assert.Equal(t, int32(1), runs.Load())
}

func TestSchedulerSurvivesFailingAndPanickingTasks(t *testing.T) {
	var after atomic.Int32
	fixed := func() time.Time { return time.Now() }
	tasks := []Task{
		{Name: "fails", Every: 10 * time.Millisecond, Run: func(context.Context) error { return errors.New("boom") }},
		{Name: "panics", Every: 10 * time.Millisecond, Run: func(context.Context) error { panic("bad") }},
		{Name: "fine", Every: 10 * time.Millisecond, Run: func(context.Context) error { after.Add(1); return nil }},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	NewScheduler(tasks, newMemLocker(), quietLogger(), fixed).Run(ctx)
	assert.Positive(t, after.Load())
}
