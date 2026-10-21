package service

import (
	"context"
	"errors"
	"path"
	"regexp"
	"strings"

	"github.com/suwahadi/seafarerscrew/backend/internal/platform/s3"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// MediaPrefix is the storage prefix of files that may be served publicly.
const MediaPrefix = "public/"

// MediaPath is the public API path that serves public files.
const MediaPath = "/api/public/v1/media/"

var mediaKeyPattern = regexp.MustCompile(`^[a-z][a-z_]{1,39}/\d{4}/\d{2}/[A-Za-z0-9]+(-\d{2,4})?\.(webp|svg)$`)

var mediaTypes = map[string]string{".webp": "image/webp", ".svg": "image/svg+xml"}

// ObjectReader downloads stored objects.
type ObjectReader interface {
	Get(ctx context.Context, key string) ([]byte, error)
}

// PublicMediaService serves CMS images, logos and flags stored under the public prefix.
type PublicMediaService struct {
	storage ObjectReader
	baseURL string
}

// NewPublicMediaService builds a PublicMediaService; baseURL is the public API origin.
func NewPublicMediaService(storage ObjectReader, baseURL string) *PublicMediaService {
	return &PublicMediaService{storage: storage, baseURL: strings.TrimRight(baseURL, "/")}
}

// URL returns the absolute public URL of a stored key, or "" for private keys.
func (s *PublicMediaService) URL(storageKey string) string {
	rest, ok := strings.CutPrefix(storageKey, MediaPrefix)
	if !ok {
		return ""
	}
	return s.baseURL + MediaPath + rest
}

// Object returns the bytes and content type of a public file addressed by its path below the prefix.
func (s *PublicMediaService) Object(ctx context.Context, p string) ([]byte, string, error) {
	if !mediaKeyPattern.MatchString(p) {
		return nil, "", apperr.NotFound("File")
	}
	data, err := s.storage.Get(ctx, MediaPrefix+p)
	if errors.Is(err, s3.ErrNotFound) {
		return nil, "", apperr.NotFound("File")
	}
	if err != nil {
		return nil, "", err
	}
	return data, mediaTypes[path.Ext(p)], nil
}
