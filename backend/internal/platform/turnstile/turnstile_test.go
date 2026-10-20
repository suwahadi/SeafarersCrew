package turnstile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifySendsFormAndParsesResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "sec", r.PostForm.Get("secret"))
		assert.Equal(t, "tok", r.PostForm.Get("response"))
		assert.Equal(t, "203.0.113.9", r.PostForm.Get("remoteip"))
		_, _ = w.Write([]byte(`{"success":true,"hostname":"example.com","error-codes":[]}`))
	}))
	defer srv.Close()
	res, err := New(srv.URL, time.Second).Verify(context.Background(), "sec", "tok", "203.0.113.9")
	require.NoError(t, err)
	assert.True(t, res.Success)
	assert.Equal(t, "example.com", res.Hostname)
}

func TestVerifyReportsErrorCodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"error-codes":["invalid-input-secret"]}`))
	}))
	defer srv.Close()
	res, err := New(srv.URL, time.Second).Verify(context.Background(), "bad", "tok", "")
	require.NoError(t, err)
	assert.False(t, res.Success)
	assert.True(t, res.HasError("invalid-input-secret"))
	assert.False(t, res.HasError("timeout-or-duplicate"))
}

func TestVerifyFailsOnInvalidResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>"))
	}))
	defer srv.Close()
	_, err := New(srv.URL, time.Second).Verify(context.Background(), "s", "t", "")
	assert.Error(t, err)
}
