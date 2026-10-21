package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type memCache struct {
	versions map[string]int
	data     map[string][]byte
}

func newMemCache() *memCache { return &memCache{versions: map[string]int{}, data: map[string][]byte{}} }

func (m *memCache) TagVersions(_ context.Context, tags []string) (string, error) {
	var b strings.Builder
	for _, t := range tags {
		b.WriteString(t)
		b.WriteByte('=')
		b.WriteByte(byte('0' + m.versions[t]))
	}
	return b.String(), nil
}

func (m *memCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	v, ok := m.data[key]
	return v, ok, nil
}

func (m *memCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.data[key] = value
	return nil
}

func countingHandler(calls *int, status int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*calls++
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"data":` + string(rune('0'+*calls)) + `}`))
	})
}

func get(h http.Handler, url string, header ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	if len(header) == 2 {
		req.Header.Set(header[0], header[1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPublicCacheServesRepeatReadsFromCache(t *testing.T) {
	calls := 0
	h := PublicCache(newMemCache(), time.Minute, "news")(countingHandler(&calls, http.StatusOK))

	first := get(h, "/news?page=1")
	second := get(h, "/news?page=1")

	assert.Equal(t, 1, calls)
	assert.Equal(t, "MISS", first.Header().Get("X-Cache"))
	assert.Equal(t, "HIT", second.Header().Get("X-Cache"))
	assert.Equal(t, first.Body.String(), second.Body.String())
	assert.Equal(t, PublicCacheControl, second.Header().Get("Cache-Control"))
	assert.NotEmpty(t, second.Header().Get("ETag"))
}

func TestPublicCacheKeysByFullURL(t *testing.T) {
	calls := 0
	h := PublicCache(newMemCache(), time.Minute, "news")(countingHandler(&calls, http.StatusOK))
	get(h, "/news?page=1")
	get(h, "/news?page=2")
	assert.Equal(t, 2, calls)
}

func TestPublicCacheMissesAfterTagBump(t *testing.T) {
	calls := 0
	store := newMemCache()
	h := PublicCache(store, time.Minute, "news", "site")(countingHandler(&calls, http.StatusOK))
	get(h, "/news")
	store.versions["site"]++
	get(h, "/news")
	assert.Equal(t, 2, calls)
}

func TestPublicCacheAnswersMatchingETagWithNotModified(t *testing.T) {
	calls := 0
	h := PublicCache(newMemCache(), time.Minute)(countingHandler(&calls, http.StatusOK))
	etag := get(h, "/site").Header().Get("ETag")
	rec := get(h, "/site", "If-None-Match", etag)
	assert.Equal(t, http.StatusNotModified, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestPublicCacheDoesNotStoreErrors(t *testing.T) {
	calls := 0
	h := PublicCache(newMemCache(), time.Minute)(countingHandler(&calls, http.StatusNotFound))
	rec := get(h, "/pages/missing")
	get(h, "/pages/missing")
	assert.Equal(t, 2, calls)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, rec.Header().Get("ETag"))
}
