package slug

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMake(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"simple title", "About Us", "about-us"},
		{"collapses repeated separators", "Crew  --  Management!!", "crew-management"},
		{"trims leading and trailing separators", "  ---Hello World---  ", "hello-world"},
		{"transliterates accents", "Café Crème Brûlée", "cafe-creme-brulee"},
		{"transliterates special letters", "Straße Øresund Łódź", "strasse-oresund-lodz"},
		{"ampersand becomes and", "Terms & Conditions", "terms-and-conditions"},
		{"drops non latin scripts", "日本 Crew 2026", "crew-2026"},
		{"keeps digits", "Top 10 Tips for 2nd Officers", "top-10-tips-for-2nd-officers"},
		{"empty input", "", ""},
		{"only symbols", "!!!", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Make(tc.in))
		})
	}
}

func TestMakeLimitsLengthWithoutTrailingHyphen(t *testing.T) {
	got := Make(strings.Repeat("abcd ", 30))
	assert.LessOrEqual(t, len(got), MaxLength)
	assert.False(t, strings.HasSuffix(got, "-"))
	assert.True(t, Valid(got))
}

func TestWithSuffix(t *testing.T) {
	assert.Equal(t, "about-us", WithSuffix("about-us", 1))
	assert.Equal(t, "about-us-2", WithSuffix("about-us", 2))
	assert.Equal(t, "about-us-13", WithSuffix("about-us", 13))

	long := Make(strings.Repeat("x", 100))
	got := WithSuffix(long, 3)
	assert.Len(t, got, MaxLength)
	assert.True(t, strings.HasSuffix(got, "-3"))
}

func TestValid(t *testing.T) {
	assert.True(t, Valid("about-us"))
	assert.False(t, Valid("About-Us"))
	assert.False(t, Valid("about--us"))
	assert.False(t, Valid("-about"))
	assert.False(t, Valid(""))
}
