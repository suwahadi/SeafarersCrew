// Package secret encrypts setting values with AES-256-GCM.
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/token"
)

// KeySize is the required key length in bytes.
const KeySize = 32

const prefix = "v1:"

// ErrInvalidKey is returned when the key is not 32 bytes.
var ErrInvalidKey = errors.New("encryption key must be exactly 32 bytes")

// ErrInvalidCiphertext is returned when decryption input is malformed or tampered.
var ErrInvalidCiphertext = errors.New("invalid ciphertext")

// Box encrypts and decrypts strings.
type Box struct {
	aead cipher.AEAD
}

// NewBox builds a Box from a 32-byte key.
func NewBox(key []byte) (*Box, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}
	return &Box{aead: aead}, nil
}

// Encrypt returns a versioned base64 ciphertext of plaintext.
func (b *Box) Encrypt(plaintext string) (string, error) {
	nonce, err := token.Bytes(b.aead.NonceSize())
	if err != nil {
		return "", err
	}
	sealed := b.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt.
func (b *Box) Decrypt(ciphertext string) (string, error) {
	if len(ciphertext) < len(prefix) || ciphertext[:len(prefix)] != prefix {
		return "", ErrInvalidCiphertext
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext[len(prefix):])
	if err != nil || len(raw) < b.aead.NonceSize() {
		return "", ErrInvalidCiphertext
	}
	nonce, sealed := raw[:b.aead.NonceSize()], raw[b.aead.NonceSize():]
	plain, err := b.aead.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	return string(plain), nil
}

// Mask hides all but the last four characters of a secret.
func Mask(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= 4 {
		return "••••"
	}
	return "••••••••" + string(runes[len(runes)-4:])
}
