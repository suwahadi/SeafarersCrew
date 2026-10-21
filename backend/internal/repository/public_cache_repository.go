package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// PublicCacheRepository stores public API responses keyed by URL and tag versions.
type PublicCacheRepository struct {
	rdb *goredis.Client
}

// NewPublicCacheRepository builds a PublicCacheRepository.
func NewPublicCacheRepository(rdb *goredis.Client) *PublicCacheRepository {
	return &PublicCacheRepository{rdb: rdb}
}

func tagKey(tag string) string { return "pubtag:" + tag }

// TagVersions returns the current versions of tags joined into one string.
func (r *PublicCacheRepository) TagVersions(ctx context.Context, tags []string) (string, error) {
	if len(tags) == 0 {
		return "", nil
	}
	keys := make([]string, len(tags))
	for i, t := range tags {
		keys[i] = tagKey(t)
	}
	vals, err := r.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return "", fmt.Errorf("read cache tag versions: %w", err)
	}
	parts := make([]string, len(vals))
	for i, v := range vals {
		s, _ := v.(string)
		parts[i] = tags[i] + "=" + s
	}
	return strings.Join(parts, ","), nil
}

// BumpTags invalidates every cached response that depends on tags.
func (r *PublicCacheRepository) BumpTags(ctx context.Context, tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	pipe := r.rdb.Pipeline()
	for _, t := range tags {
		pipe.Incr(ctx, tagKey(t))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("bump cache tags: %w", err)
	}
	return nil
}

// Get returns a cached response.
func (r *PublicCacheRepository) Get(ctx context.Context, key string) ([]byte, bool, error) {
	b, err := r.rdb.Get(ctx, "pubcache:"+key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("public cache get: %w", err)
	}
	return b, true, nil
}

// Set stores a response for ttl.
func (r *PublicCacheRepository) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := r.rdb.Set(ctx, "pubcache:"+key, value, ttl).Err(); err != nil {
		return fmt.Errorf("public cache set: %w", err)
	}
	return nil
}
