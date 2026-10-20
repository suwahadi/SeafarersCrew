package request

import (
	"net/url"
	"strconv"
	"strings"
)

// Page sizes allowed by the datagrid.
var allowedPageSizes = map[int]bool{10: true, 20: true, 30: true, 50: true, 100: true}

// DefaultPageSize is used when per_page is missing or not allowed.
const DefaultPageSize = 20

// ListParams holds the common list query parameters.
type ListParams struct {
	Page    int
	PerPage int
	Sort    string
	Desc    bool
	Q       string
}

// Offset returns the row offset of the page.
func (p ListParams) Offset() int { return (p.Page - 1) * p.PerPage }

// ParseList reads page, per_page, sort, order and q from a query string.
func ParseList(q url.Values) ListParams {
	p := ListParams{Page: 1, PerPage: DefaultPageSize}
	if n, err := strconv.Atoi(q.Get("page")); err == nil && n > 0 && n <= 1_000_000 {
		p.Page = n
	}
	if n, err := strconv.Atoi(q.Get("per_page")); err == nil && allowedPageSizes[n] {
		p.PerPage = n
	}
	p.Sort = strings.TrimSpace(q.Get("sort"))
	p.Desc = strings.EqualFold(q.Get("order"), "desc")
	p.Q = strings.TrimSpace(q.Get("q"))
	if r := []rune(p.Q); len(r) > 100 {
		p.Q = string(r[:100])
	}
	return p
}

// Bool parses an optional boolean query parameter.
func Bool(q url.Values, key string) *bool {
	switch strings.ToLower(q.Get(key)) {
	case "true", "1":
		v := true
		return &v
	case "false", "0":
		v := false
		return &v
	}
	return nil
}

// Int parses an optional integer query parameter.
func Int(q url.Values, key string) *int {
	n, err := strconv.Atoi(q.Get(key))
	if err != nil {
		return nil
	}
	return &n
}

// CSV parses a comma-separated query parameter into trimmed non-empty values.
func CSV(q url.Values, key string) []string {
	var out []string
	for _, raw := range q[key] {
		for _, v := range strings.Split(raw, ",") {
			if v = strings.TrimSpace(v); v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}
