// Package password hashes and verifies secrets with Argon2id in PHC string format.
package password

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/token"
)

// Params controls the Argon2id cost.
type Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

const keyLength = 32

// DefaultParams follows the OWASP recommendation with extra memory headroom.
var DefaultParams = Params{Memory: 64 * 1024, Iterations: 3, Parallelism: 2, SaltLength: 16, KeyLength: keyLength}

// ErrMalformedHash is returned when a stored hash cannot be parsed.
var ErrMalformedHash = errors.New("malformed argon2id hash")

// Hasher hashes and verifies secrets with fixed parameters.
type Hasher struct {
	params Params
}

// NewHasher returns a Hasher using p; the derived key is always 32 bytes.
func NewHasher(p Params) *Hasher {
	p.KeyLength = keyLength
	return &Hasher{params: p}
}

// Hash returns the PHC-encoded Argon2id hash of secret.
func (h *Hasher) Hash(secret string) (string, error) {
	salt, err := token.Bytes(int(h.params.SaltLength))
	if err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	key := argon2.IDKey([]byte(secret), salt, h.params.Iterations, h.params.Memory, h.params.Parallelism, h.params.KeyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.params.Memory, h.params.Iterations, h.params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

// Verify reports whether secret matches the encoded hash.
func (h *Hasher) Verify(secret, encoded string) (bool, error) {
	p, salt, key, err := decode(encoded)
	if err != nil {
		return false, err
	}
	other := argon2.IDKey([]byte(secret), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	return subtle.ConstantTimeCompare(key, other) == 1, nil
}

func decode(encoded string) (Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrMalformedHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return Params{}, nil, nil, ErrMalformedHash
	}
	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism); err != nil {
		return Params{}, nil, nil, ErrMalformedHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Params{}, nil, nil, ErrMalformedHash
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(key) != keyLength {
		return Params{}, nil, nil, ErrMalformedHash
	}
	p.KeyLength = keyLength
	return p, salt, key, nil
}
