package token

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileNameIsTwelveLowercaseAlphanumerics(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-z0-9]{12}$`)
	for range 500 {
		name, err := FileName()
		require.NoError(t, err)
		assert.Regexp(t, pattern, name)
	}
}

func TestRecoveryCodeFormat(t *testing.T) {
	code, err := RecoveryCode()
	require.NoError(t, err)
	assert.Regexp(t, `^[2-9A-HJKMNP-Z]{5}-[2-9A-HJKMNP-Z]{5}$`, code)
}

func TestNormalizeRecoveryCode(t *testing.T) {
	assert.Equal(t, "ABCDE23456", NormalizeRecoveryCode(" abcde-23456 "))
	assert.Equal(t, "ABCDE23456", NormalizeRecoveryCode("ABCDE 23456"))
}

func TestURLSafeLength(t *testing.T) {
	s, err := URLSafe(32)
	require.NoError(t, err)
	assert.Len(t, s, 43)
}

func TestSHA256Hex(t *testing.T) {
	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", SHA256Hex("hello"))
}
