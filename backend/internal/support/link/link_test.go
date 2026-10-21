package link

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidAcceptsOnlyRelativeHTTPSMailtoAndTel(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"/career", true},
		{"/news/crew-change#top", true},
		{"https://example.com/a?b=c", true},
		{"mailto:crew@example.com", true},
		{"tel:+628123456789", true},
		{"http://example.com", false},
		{"//evil.example.com", false},
		{`/\evil.example.com`, false},
		{"javascript:alert(1)", false},
		{"data:text/html,x", false},
		{"career", false},
		{"https:///nohost", false},
		{"mailto:nobody", false},
		{"tel:abc", false},
		{"/with space", false},
		{"", false},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assert.Equal(t, c.want, Valid(c.in))
		})
	}
}

func TestHTTPSRejectsPlainHTTPAndRelative(t *testing.T) {
	assert.True(t, HTTPS("https://client.example.com"))
	assert.False(t, HTTPS("http://client.example.com"))
	assert.False(t, HTTPS("/client"))
	assert.False(t, HTTPS("https://"))
}
