package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type mapRedirects map[string]string

func (m mapRedirects) Resolve(_ context.Context, path string) (model.Redirect, error) {
	if to, ok := m[path]; ok {
		return model.Redirect{FromPath: path, ToPath: to, StatusCode: 301}, nil
	}
	return model.Redirect{}, repository.ErrNotFound
}

func TestRedirectServiceNormalizesPathSlashes(t *testing.T) {
	s := NewRedirectService(mapRedirects{"/news/old-title": "/news/new-title"})
	rd, err := s.Resolve(context.Background(), "news/old-title/")
	require.NoError(t, err)
	assert.Equal(t, "/news/new-title", rd.ToPath)
}

func TestRedirectServiceReportsUnknownPathsAsNotFound(t *testing.T) {
	_, err := NewRedirectService(mapRedirects{}).Resolve(context.Background(), "missing")
	var ae *apperr.Error
	require.True(t, errors.As(err, &ae))
	assert.Equal(t, http.StatusNotFound, ae.Status)
}
