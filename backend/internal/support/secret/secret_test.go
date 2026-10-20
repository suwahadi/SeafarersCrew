package secret

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBox(t *testing.T) *Box {
	t.Helper()
	b, err := NewBox(bytes.Repeat([]byte{7}, KeySize))
	require.NoError(t, err)
	return b
}

func TestNewBoxRejectsWrongKeyLength(t *testing.T) {
	for _, n := range []int{0, 16, 24, 31, 33} {
		_, err := NewBox(make([]byte, n))
		assert.ErrorIs(t, err, ErrInvalidKey)
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	b := newBox(t)
	ct, err := b.Encrypt("xkeysib-abc123")
	require.NoError(t, err)
	assert.NotContains(t, ct, "xkeysib")

	pt, err := b.Decrypt(ct)
	require.NoError(t, err)
	assert.Equal(t, "xkeysib-abc123", pt)
}

func TestEncryptUsesFreshNonce(t *testing.T) {
	b := newBox(t)
	a, _ := b.Encrypt("same")
	c, _ := b.Encrypt("same")
	assert.NotEqual(t, a, c)
}

func TestDecryptRejectsTamperedOrForeignCiphertext(t *testing.T) {
	b := newBox(t)
	ct, err := b.Encrypt("value")
	require.NoError(t, err)

	tampered := []byte(ct)
	tampered[len(tampered)-2] ^= 1
	_, err = b.Decrypt(string(tampered))
	assert.ErrorIs(t, err, ErrInvalidCiphertext)

	other, err := NewBox(bytes.Repeat([]byte{9}, KeySize))
	require.NoError(t, err)
	_, err = other.Decrypt(ct)
	assert.ErrorIs(t, err, ErrInvalidCiphertext)

	for _, bad := range []string{"", "plain", "v1:!!!", "v1:AAAA"} {
		_, err = b.Decrypt(bad)
		assert.ErrorIs(t, err, ErrInvalidCiphertext, bad)
	}
}

func TestMask(t *testing.T) {
	assert.Equal(t, "", Mask(""))
	assert.Equal(t, "••••", Mask("abcd"))
	assert.Equal(t, "••••••••6789", Mask("xkeysib-123456789"))
}
