package request

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseList(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  ListParams
	}{
		{"defaults", "", ListParams{Page: 1, PerPage: 20}},
		{"allowed size", "page=3&per_page=50&sort=full_name&order=DESC&q=+john+", ListParams{Page: 3, PerPage: 50, Sort: "full_name", Desc: true, Q: "john"}},
		{"disallowed size falls back", "per_page=25", ListParams{Page: 1, PerPage: 20}},
		{"negative page", "page=-2", ListParams{Page: 1, PerPage: 20}},
		{"garbage page", "page=abc", ListParams{Page: 1, PerPage: 20}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, _ := url.ParseQuery(tc.query)
			assert.Equal(t, tc.want, ParseList(q))
		})
	}
}

func TestParseListTruncatesSearch(t *testing.T) {
	q := url.Values{"q": {strings.Repeat("é", 150)}}
	assert.Len(t, []rune(ParseList(q).Q), 100)
}

func TestOffset(t *testing.T) {
	assert.Equal(t, 40, ListParams{Page: 3, PerPage: 20}.Offset())
}

func TestCSVAndBool(t *testing.T) {
	q, _ := url.ParseQuery("status=a,b&status=c&active=true&x=0")
	assert.Equal(t, []string{"a", "b", "c"}, CSV(q, "status"))
	assert.True(t, *Bool(q, "active"))
	assert.False(t, *Bool(q, "x"))
	assert.Nil(t, Bool(q, "missing"))
}
