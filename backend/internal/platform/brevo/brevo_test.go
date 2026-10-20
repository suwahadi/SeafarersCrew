package brevo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendPostsMessageWithAttachments(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/smtp/email", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "key-123", r.Header.Get("api-key"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"messageId":"<abc@smtp-relay>"}`))
	}))
	defer srv.Close()

	id, err := New(srv.URL, time.Second).Send(context.Background(), Message{
		APIKey:      "key-123",
		Sender:      Address{Email: "no-reply@example.com", Name: "Seafarers Crew"},
		To:          []Address{{Email: "a@example.com"}},
		CC:          []Address{{Email: "b@example.com"}},
		ReplyTo:     &Address{Email: "ops@example.com"},
		Subject:     "Contract",
		HTML:        "<p>Hi</p>",
		Attachments: []Attachment{{Name: "sea.pdf", Content: []byte("%PDF-1.7")}},
	})
	require.NoError(t, err)
	assert.Equal(t, "<abc@smtp-relay>", id)
	assert.Equal(t, "Contract", got["subject"])
	assert.Equal(t, "<p>Hi</p>", got["htmlContent"])
	assert.Equal(t, map[string]any{"email": "ops@example.com"}, got["replyTo"])
	att := got["attachment"].([]any)[0].(map[string]any)
	assert.Equal(t, "sea.pdf", att["name"])
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("%PDF-1.7")), att["content"])
}

func TestErrorsAreClassified(t *testing.T) {
	cases := []struct {
		status    int
		retryable bool
	}{
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(`{"code":"invalid_parameter","message":"bad things"}`))
		}))
		_, err := New(srv.URL, time.Second).Send(context.Background(), Message{APIKey: "k"})
		srv.Close()
		require.Error(t, err)
		assert.Equal(t, tc.retryable, IsRetryable(err), tc.status)
		assert.Contains(t, err.Error(), "bad things")
	}
}

func TestNetworkErrorIsRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()
	_, err := New(url, time.Second).Send(context.Background(), Message{APIKey: "k"})
	require.Error(t, err)
	assert.True(t, IsRetryable(err))
}

func TestMissingAPIKeyIsPermanent(t *testing.T) {
	_, err := New("http://unused", time.Second).Send(context.Background(), Message{})
	require.Error(t, err)
	assert.False(t, IsRetryable(err))
	_, err = New("http://unused", time.Second).Account(context.Background(), "")
	assert.False(t, IsRetryable(err))
}

func TestAccountReturnsOwnerEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/account", r.URL.Path)
		_, _ = w.Write([]byte(`{"email":"owner@example.com"}`))
	}))
	defer srv.Close()
	email, err := New(srv.URL, time.Second).Account(context.Background(), "k")
	require.NoError(t, err)
	assert.Equal(t, "owner@example.com", email)
}

func TestNonJSONErrorBodyIsTruncated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write(make([]byte, 0))
	}))
	defer srv.Close()
	_, err := New(srv.URL, time.Second).Send(context.Background(), Message{APIKey: "k"})
	assert.ErrorContains(t, err, "empty response")
}
