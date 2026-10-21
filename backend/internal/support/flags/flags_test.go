package flags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKnownAcceptsLowercaseISOCodesOnly(t *testing.T) {
	for _, code := range []string{"id", "gb", "jp", "cn", "za"} {
		assert.True(t, Known(code), code)
	}
	for _, code := range []string{"", "GB", "xx", "uk", "gbr", "i"} {
		assert.False(t, Known(code), code)
	}
}
