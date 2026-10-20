// Package publicid generates the 9-character public identifiers shown to users.
package publicid

import (
	"crypto/rand"
	"fmt"
	"io"
)

// Alphabet holds uppercase letters and digits without the look-alikes 0, O, 1, I and L.
const Alphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// Length is the fixed size of every public ID.
const Length = 9

// MaxAttempts is how many times a caller retries insertion after a unique-index conflict.
const MaxAttempts = 5

// New returns a random public ID using crypto/rand.
func New() (string, error) {
	return NewFrom(rand.Reader)
}

// NewFrom returns a random public ID read from the given entropy source.
func NewFrom(r io.Reader) (string, error) {
	return randomString(r, Alphabet, Length)
}

// Valid reports whether s is a well-formed public ID.
func Valid(s string) bool {
	if len(s) != Length {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !inAlphabet(s[i]) {
			return false
		}
	}
	return true
}

func inAlphabet(c byte) bool {
	for i := 0; i < len(Alphabet); i++ {
		if Alphabet[i] == c {
			return true
		}
	}
	return false
}

func randomString(r io.Reader, alphabet string, length int) (string, error) {
	limit := 256 - 256%len(alphabet)
	out := make([]byte, 0, length)
	buf := make([]byte, length*2)
	for len(out) < length {
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", fmt.Errorf("read random bytes: %w", err)
		}
		for _, b := range buf {
			if int(b) >= limit {
				continue
			}
			out = append(out, alphabet[int(b)%len(alphabet)])
			if len(out) == length {
				break
			}
		}
	}
	return string(out), nil
}
