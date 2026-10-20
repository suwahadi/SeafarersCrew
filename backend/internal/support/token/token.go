// Package token creates random secrets and file names from crypto/rand.
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/publicid"
)

const fileNameAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// FileNameLength is the length of random storage names such as n87hds5300gx.
const FileNameLength = 12

// Bytes returns n random bytes.
func Bytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, fmt.Errorf("read random bytes: %w", err)
	}
	return b, nil
}

// URLSafe returns n random bytes encoded as unpadded base64url.
func URLSafe(n int) (string, error) {
	b, err := Bytes(n)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// FileName returns a random lowercase alphanumeric name of FileNameLength characters.
func FileName() (string, error) {
	return pick(fileNameAlphabet, FileNameLength)
}

// RecoveryCode returns a one-time code formatted as XXXXX-XXXXX.
func RecoveryCode() (string, error) {
	raw, err := pick(publicid.Alphabet, 10)
	if err != nil {
		return "", err
	}
	return raw[:5] + "-" + raw[5:], nil
}

// NormalizeRecoveryCode uppercases a code and strips spaces and hyphens.
func NormalizeRecoveryCode(code string) string {
	r := strings.NewReplacer("-", "", " ", "")
	return strings.ToUpper(r.Replace(strings.TrimSpace(code)))
}

// SHA256Hex returns the lowercase hex SHA-256 digest of s.
func SHA256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func pick(alphabet string, length int) (string, error) {
	limit := 256 - 256%len(alphabet)
	out := make([]byte, 0, length)
	for len(out) < length {
		buf, err := Bytes(length)
		if err != nil {
			return "", err
		}
		for _, b := range buf {
			if int(b) >= limit || len(out) == length {
				continue
			}
			out = append(out, alphabet[int(b)%len(alphabet)])
		}
	}
	return string(out), nil
}
