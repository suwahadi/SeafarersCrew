package service

import (
	"context"
	"errors"
	"strings"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// RedirectResolver reads stored redirects.
type RedirectResolver interface {
	Resolve(ctx context.Context, path string) (model.Redirect, error)
}

// RedirectService resolves legacy public paths.
type RedirectService struct {
	store RedirectResolver
}

// NewRedirectService builds a RedirectService.
func NewRedirectService(store RedirectResolver) *RedirectService {
	return &RedirectService{store: store}
}

// Resolve returns where path now lives.
func (s *RedirectService) Resolve(ctx context.Context, path string) (model.Redirect, error) {
	path = "/" + strings.Trim(path, "/")
	rd, err := s.store.Resolve(ctx, path)
	if errors.Is(err, repository.ErrNotFound) {
		return rd, apperr.NotFound("Redirect")
	}
	return rd, err
}
