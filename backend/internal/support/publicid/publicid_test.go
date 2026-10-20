package publicid

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlphabetExcludesLookAlikes(t *testing.T) {
	assert.Len(t, Alphabet, 31)
	for _, c := range "0O1IL" {
		assert.NotContains(t, Alphabet, string(c))
	}
}

func TestNewReturnsNineCharactersFromAlphabet(t *testing.T) {
	for range 1000 {
		id, err := New()
		require.NoError(t, err)
		assert.Len(t, id, Length)
		assert.True(t, Valid(id), id)
	}
}

func TestNewProducesDistinctIDs(t *testing.T) {
	seen := make(map[string]struct{}, 10000)
	for range 10000 {
		id, err := New()
		require.NoError(t, err)
		seen[id] = struct{}{}
	}
	assert.Len(t, seen, 10000)
}

func TestNewFromRejectsBiasedBytes(t *testing.T) {
	// 248 is the first byte above the unbiased limit (256 - 256%31) and must be skipped.
	src := bytes.NewReader(append(bytes.Repeat([]byte{255}, 18), bytes.Repeat([]byte{0}, 18)...))
	id, err := NewFrom(src)
	require.NoError(t, err)
	assert.Equal(t, strings.Repeat("2", Length), id)
}

func TestNewFromPropagatesReadErrors(t *testing.T) {
	_, err := NewFrom(failingReader{})
	assert.Error(t, err)
}

func TestValid(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"example from PRD", "SR6H70QU9", false},
		{"allowed characters", "SR6H72QU9", true},
		{"too short", "SR6H7", false},
		{"too long", "SR6H72QU9A", false},
		{"lowercase", "sr6h72qu9", false},
		{"contains I", "SR6H72QI9", false},
		{"contains L", "SR6H72QL9", false},
		{"contains 1", "SR6H72Q19", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Valid(tc.in))
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }
