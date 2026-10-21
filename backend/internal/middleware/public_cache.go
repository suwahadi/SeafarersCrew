package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/logctx"
)

// PublicCacheControl is the browser and CDN policy for public API reads.
const PublicCacheControl = "public, max-age=60, stale-while-revalidate=300"

// PublicCacheStore keeps public responses keyed by URL and tag versions.
type PublicCacheStore interface {
	TagVersions(ctx context.Context, tags []string) (string, error)
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// PublicCache serves GET responses from the cache until ttl passes or one of tags is bumped,
// and answers conditional requests with 304.
func PublicCache(store PublicCacheStore, ttl time.Duration, tags ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}
			ctx := r.Context()
			versions, err := store.TagVersions(ctx, tags)
			if err != nil {
				logctx.From(ctx).WarnContext(ctx, "public cache bypassed", "error", err)
				next.ServeHTTP(w, r)
				return
			}
			key := cacheKey(r.URL.RequestURI(), versions)
			if body, ok, err := store.Get(ctx, key); err == nil && ok {
				writeCached(w, r, body, "HIT")
				return
			}
			rec := &bufferedResponse{header: http.Header{}, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			if rec.status != http.StatusOK {
				rec.flush(w)
				return
			}
			body := rec.body.Bytes()
			if err := store.Set(ctx, key, body, ttl); err != nil {
				logctx.From(ctx).WarnContext(ctx, "public cache not stored", "error", err)
			}
			for k, v := range rec.header {
				w.Header()[k] = v
			}
			writeCached(w, r, body, "MISS")
		})
	}
}

func cacheKey(uri, versions string) string {
	sum := sha256.Sum256([]byte(uri + "|" + versions))
	return hex.EncodeToString(sum[:])
}

func etagOf(body []byte) string {
	sum := sha256.Sum256(body)
	return `"` + hex.EncodeToString(sum[:12]) + `"`
}

func writeCached(w http.ResponseWriter, r *http.Request, body []byte, state string) {
	etag := etagOf(body)
	h := w.Header()
	h.Set("Cache-Control", PublicCacheControl)
	h.Set("ETag", etag)
	h.Set("X-Cache", state)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	h.Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (b *bufferedResponse) Header() http.Header { return b.header }

func (b *bufferedResponse) WriteHeader(code int) { b.status = code }

func (b *bufferedResponse) Write(p []byte) (int, error) { return b.body.Write(p) }

func (b *bufferedResponse) flush(w http.ResponseWriter) {
	for k, v := range b.header {
		w.Header()[k] = v
	}
	w.WriteHeader(b.status)
	_, _ = w.Write(b.body.Bytes())
}
