package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// CacheRepository stores cached payloads in Redis.
type CacheRepository struct {
	rdb *goredis.Client
}

// NewCacheRepository returns a CacheRepository.
func NewCacheRepository(rdb *goredis.Client) *CacheRepository {
	return &CacheRepository{rdb: rdb}
}

// Get returns the cached value and whether it was found.
func (r *CacheRepository) Get(ctx context.Context, key string) ([]byte, bool, error) {
	b, err := r.rdb.Get(ctx, "cache:"+key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("cache get: %w", err)
	}
	return b, true, nil
}

// Set stores value under key for ttl.
func (r *CacheRepository) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := r.rdb.Set(ctx, "cache:"+key, value, ttl).Err(); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
	return nil
}

// Delete removes keys.
func (r *CacheRepository) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	prefixed := make([]string, len(keys))
	for i, k := range keys {
		prefixed[i] = "cache:" + k
	}
	if err := r.rdb.Del(ctx, prefixed...).Err(); err != nil {
		return fmt.Errorf("cache delete: %w", err)
	}
	return nil
}
