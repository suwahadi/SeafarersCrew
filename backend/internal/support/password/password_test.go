package password

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fastParams = Params{Memory: 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}

func TestHashProducesArgon2idPHCString(t *testing.T) {
	h := NewHasher(fastParams)
	encoded, err := h.Hash("correct horse battery staple")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(encoded, "$argon2id$v=19$m=1024,t=1,p=1$"))
}

func TestHashUsesUniqueSalt(t *testing.T) {
	h := NewHasher(fastParams)
	a, err := h.Hash("same")
	require.NoError(t, err)
	b, err := h.Hash("same")
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

func TestVerify(t *testing.T) {
	h := NewHasher(fastParams)
	encoded, err := h.Hash("s3cret!")
	require.NoError(t, err)

	ok, err := h.Verify("s3cret!", encoded)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = h.Verify("wrong", encoded)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestVerifyUsesParamsFromHash(t *testing.T) {
	encoded, err := NewHasher(fastParams).Hash("pw")
	require.NoError(t, err)
	ok, err := NewHasher(DefaultParams).Verify("pw", encoded)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerifyRejectsMalformedHashes(t *testing.T) {
	h := NewHasher(fastParams)
	for _, bad := range []string{"", "plain", "$argon2i$v=19$m=1,t=1,p=1$a$b", "$argon2id$v=18$m=1,t=1,p=1$YQ$Yg", "$argon2id$v=19$m=x$YQ$Yg", "$argon2id$v=19$m=1,t=1,p=1$!!$Yg"} {
		_, err := h.Verify("pw", bad)
		assert.ErrorIs(t, err, ErrMalformedHash, bad)
	}
}
