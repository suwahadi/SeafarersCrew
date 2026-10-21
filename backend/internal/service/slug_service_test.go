package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func takenSet(used ...string) func(context.Context, string) (bool, error) {
	set := map[string]bool{}
	for _, u := range used {
		set[u] = true
	}
	return func(_ context.Context, s string) (bool, error) { return set[s], nil }
}

func TestUniqueSlugComesFromTitleWhenNotGiven(t *testing.T) {
	s, err := UniqueSlug(context.Background(), "Crew Change in Singapore!", "", takenSet())
	require.NoError(t, err)
	assert.Equal(t, "crew-change-in-singapore", s)
}

func TestUniqueSlugNormalizesAnEditedSlug(t *testing.T) {
	s, err := UniqueSlug(context.Background(), "ignored", "  Über Uns  ", takenSet())
	require.NoError(t, err)
	assert.Equal(t, "uber-uns", s)
}

func TestUniqueSlugAddsNumericSuffixOnConflict(t *testing.T) {
	s, err := UniqueSlug(context.Background(), "About", "", takenSet("about", "about-2"))
	require.NoError(t, err)
	assert.Equal(t, "about-3", s)
}

func TestUniqueSlugRejectsTitlesWithoutLettersOrDigits(t *testing.T) {
	_, err := UniqueSlug(context.Background(), "!!!", "", takenSet())
	assert.Error(t, err)
}
