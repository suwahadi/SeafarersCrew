package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/platform/s3"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type mapReader map[string][]byte

func (m mapReader) Get(_ context.Context, key string) ([]byte, error) {
	if v, ok := m[key]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("read %s: %w", key, s3.ErrNotFound)
}

func TestPublicMediaURLOnlyExposesPublicKeys(t *testing.T) {
	s := NewPublicMediaService(mapReader{}, "https://api.example.com/")
	assert.Equal(t, "https://api.example.com/api/public/v1/media/news/2026/10/Ab12Cd-960.webp", s.URL("public/news/2026/10/Ab12Cd-960.webp"))
	assert.Empty(t, s.URL("crews/2026/10/passport.pdf"))
}

func TestPublicMediaServesStoredPublicImages(t *testing.T) {
	s := NewPublicMediaService(mapReader{"public/news/2026/10/Ab12Cd.webp": []byte("RIFF"), "public/flag/2026/10/Xy9.svg": []byte("<svg/>")}, "")
	data, ct, err := s.Object(context.Background(), "news/2026/10/Ab12Cd.webp")
	require.NoError(t, err)
	assert.Equal(t, "RIFF", string(data))
	assert.Equal(t, "image/webp", ct)

	_, ct, err = s.Object(context.Background(), "flag/2026/10/Xy9.svg")
	require.NoError(t, err)
	assert.Equal(t, "image/svg+xml", ct)
}

func TestPublicMediaRejectsTraversalPrivateTypesAndMissingFiles(t *testing.T) {
	s := NewPublicMediaService(mapReader{"public/../crews/a.pdf": []byte("x")}, "")
	for _, p := range []string{"../crews/a.pdf", "news/2026/10/a.pdf", "news/2026/10/../../x.webp", "news/2026/10/missing.webp", ""} {
		_, _, err := s.Object(context.Background(), p)
		var ae *apperr.Error
		require.True(t, errors.As(err, &ae), p)
		assert.Equal(t, http.StatusNotFound, ae.Status, p)
	}
}
