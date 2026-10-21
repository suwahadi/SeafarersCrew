package service

import (
	"context"
	"fmt"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/slug"
)

const maxSlugAttempts = 200

// UniqueSlug normalizes requested (or title when requested is empty) and appends -2, -3 … until taken reports it free.
func UniqueSlug(ctx context.Context, title, requested string, taken func(ctx context.Context, s string) (bool, error)) (string, error) {
	base := slug.Make(requested)
	if base == "" {
		base = slug.Make(title)
	}
	if base == "" {
		return "", apperr.Validation(map[string]string{"slug": "Enter a title or slug that contains letters or digits."})
	}
	for n := 1; n <= maxSlugAttempts; n++ {
		candidate := slug.WithSuffix(base, n)
		used, err := taken(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !used {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no free slug for %q after %d attempts", base, maxSlugAttempts)
}
