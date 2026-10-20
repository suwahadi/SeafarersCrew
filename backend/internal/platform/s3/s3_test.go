package s3

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFakeStorage runs an in-memory S3 server; sourceCfg lets tests change Settings between calls.
func newFakeStorage(t *testing.T, bucket string) (*Storage, *Config) {
	t.Helper()
	backend := s3mem.New()
	require.NoError(t, backend.CreateBucket(bucket))
	srv := httptest.NewServer(gofakes3.New(backend).Server())
	t.Cleanup(srv.Close)
	cfg := &Config{Endpoint: srv.URL, Region: "us-east-1", Bucket: bucket, AccessKeyID: "key", SecretAccessKey: "secret", UsePathStyle: true, PresignMinutes: 5}
	return New(func(context.Context) (Config, error) { return *cfg, nil }), cfg
}

func TestPutGetStatDelete(t *testing.T) {
	st, _ := newFakeStorage(t, "seafarers")
	ctx := context.Background()
	require.NoError(t, st.Put(ctx, "crews/2026/10/abc123def456.pdf", []byte("%PDF-1.7"), "application/pdf"))

	info, err := st.Stat(ctx, "crews/2026/10/abc123def456.pdf")
	require.NoError(t, err)
	assert.Equal(t, int64(8), info.Size)
	assert.Equal(t, "application/pdf", info.ContentType)

	data, err := st.Get(ctx, "crews/2026/10/abc123def456.pdf")
	require.NoError(t, err)
	assert.Equal(t, "%PDF-1.7", string(data))

	require.NoError(t, st.Delete(ctx, "crews/2026/10/abc123def456.pdf"))
	_, err = st.Stat(ctx, "crews/2026/10/abc123def456.pdf")
	assert.Error(t, err)
}

func TestPresignGetIsShortLivedAndDownloadable(t *testing.T) {
	st, _ := newFakeStorage(t, "seafarers")
	ctx := context.Background()
	require.NoError(t, st.Put(ctx, "k.pdf", []byte("%PDF-1.7"), "application/pdf"))

	raw, lifetime, err := st.PresignGet(ctx, "k.pdf", `passport "scan".pdf`, false)
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, lifetime)
	u, err := url.Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, "300", u.Query().Get("X-Amz-Expires"))
	assert.Contains(t, u.Query().Get("response-content-disposition"), "attachment")
	assert.NotContains(t, u.Query().Get("response-content-disposition"), `"scan"`)

	res, err := http.Get(raw)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	body, _ := io.ReadAll(res.Body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "%PDF-1.7", string(body))

	inline, _, err := st.PresignGet(ctx, "k.webp", "photo.webp", true)
	require.NoError(t, err)
	iu, _ := url.Parse(inline)
	assert.Contains(t, iu.Query().Get("response-content-disposition"), "inline")
}

func TestTestChecksBucketAndWriteAccess(t *testing.T) {
	st, cfg := newFakeStorage(t, "seafarers")
	require.NoError(t, st.Test(context.Background()))

	cfg.Bucket = "missing"
	assert.ErrorContains(t, st.Test(context.Background()), "does not exist")
}

func TestStorageRequiresConfiguration(t *testing.T) {
	st := New(func(context.Context) (Config, error) { return Config{Bucket: "b"}, nil })
	assert.ErrorIs(t, st.Put(context.Background(), "k", nil, "text/plain"), ErrNotConfigured)
}

func TestClientIsRebuiltWhenSettingsChange(t *testing.T) {
	st, cfg := newFakeStorage(t, "seafarers")
	ctx := context.Background()
	first, _, err := st.current(ctx)
	require.NoError(t, err)
	again, _, err := st.current(ctx)
	require.NoError(t, err)
	assert.Same(t, first, again)

	cfg.PresignMinutes = 10
	changed, _, err := st.current(ctx)
	require.NoError(t, err)
	assert.NotSame(t, first, changed)
}
