package repository

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// LockRepository hands out short-lived locks backed by Redis.
type LockRepository struct {
	rdb *goredis.Client
}

// NewLockRepository builds a LockRepository.
func NewLockRepository(rdb *goredis.Client) *LockRepository {
	return &LockRepository{rdb: rdb}
}

// Acquire reports whether the caller is the first to take key within ttl.
func (r *LockRepository) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	ok, err := r.rdb.SetNX(ctx, "lock:"+key, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("acquire lock %s: %w", key, err)
	}
	return ok, nil
}
