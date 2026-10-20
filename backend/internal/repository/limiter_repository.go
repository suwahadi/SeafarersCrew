package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// LimiterRepository keeps fixed-window counters in Redis.
type LimiterRepository struct {
	rdb *goredis.Client
}

// NewLimiterRepository returns a LimiterRepository.
func NewLimiterRepository(rdb *goredis.Client) *LimiterRepository {
	return &LimiterRepository{rdb: rdb}
}

var incrWithWindow = goredis.NewScript(`
local n = redis.call("INCR", KEYS[1])
if n == 1 then redis.call("PEXPIRE", KEYS[1], ARGV[1]) end
return {n, redis.call("PTTL", KEYS[1])}
`)

// Hit increments the counter for key within window and returns the count and remaining window.
func (r *LimiterRepository) Hit(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	res, err := incrWithWindow.Run(ctx, r.rdb, []string{"rl:" + key}, window.Milliseconds()).Int64Slice()
	if err != nil {
		return 0, 0, fmt.Errorf("rate limit hit: %w", err)
	}
	return res[0], time.Duration(res[1]) * time.Millisecond, nil
}

// Count returns the current counter for key and its remaining window.
func (r *LimiterRepository) Count(ctx context.Context, key string) (int64, time.Duration, error) {
	pipe := r.rdb.Pipeline()
	get := pipe.Get(ctx, "rl:"+key)
	ttl := pipe.PTTL(ctx, "rl:"+key)
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, goredis.Nil) {
		return 0, 0, fmt.Errorf("rate limit count: %w", err)
	}
	n, err := get.Int64()
	if errors.Is(err, goredis.Nil) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, fmt.Errorf("rate limit count: %w", err)
	}
	return n, ttl.Val(), nil
}

// Extend sets the remaining window of key.
func (r *LimiterRepository) Extend(ctx context.Context, key string, window time.Duration) error {
	if err := r.rdb.PExpire(ctx, "rl:"+key, window).Err(); err != nil {
		return fmt.Errorf("rate limit extend: %w", err)
	}
	return nil
}

// Reset clears the counter for key.
func (r *LimiterRepository) Reset(ctx context.Context, key string) error {
	if err := r.rdb.Del(ctx, "rl:"+key).Err(); err != nil {
		return fmt.Errorf("rate limit reset: %w", err)
	}
	return nil
}
