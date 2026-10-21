package nextjs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRevalidatePostsTagsWithSecret(t *testing.T) {
	var got struct {
		Tags []string `json:"tags"`
	}
	var secret, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		secret = r.Header.Get(SecretHeader)
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL+"/", "s3cret", time.Second)
	require.NoError(t, c.Revalidate(context.Background(), []string{"news", "home"}))
	assert.Equal(t, "/api/revalidate", path)
	assert.Equal(t, "s3cret", secret)
	assert.Equal(t, []string{"news", "home"}, got.Tags)
}

func TestRevalidateReportsServerErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	defer srv.Close()
	err := NewClient(srv.URL, "x", time.Second).Revalidate(context.Background(), []string{"site"})
	assert.ErrorContains(t, err, "status 401")
}

func TestRevalidateIsSkippedWithoutWebsiteURL(t *testing.T) {
	c := NewClient("", "x", time.Second)
	assert.False(t, c.Enabled())
	assert.NoError(t, c.Revalidate(context.Background(), []string{"site"}))
}
