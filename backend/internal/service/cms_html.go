package service

import (
	"regexp"

	"github.com/microcosm-cc/bluemonday"
)

// HTMLSanitizer cleans untrusted HTML.
type HTMLSanitizer interface {
	Sanitize(s string) string
}

// NewCMSPolicy returns the HTML policy for website content and builder blocks.
func NewCMSPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("h1", "h2", "h3", "h4", "h5", "h6", "p", "br", "hr", "div", "span", "blockquote", "pre", "code",
		"strong", "b", "em", "i", "u", "s", "sub", "sup", "small", "mark", "ul", "ol", "li", "figure", "figcaption")
	p.AllowTables()
	p.AllowRelativeURLs(true)
	p.AllowURLSchemes("http", "https", "mailto", "tel")
	p.AllowAttrs("href").Matching(regexp.MustCompile(`^(https://|mailto:|tel:|/([^/\\]|$)|#)\S*$`)).OnElements("a")
	p.AllowAttrs("title").OnElements("a", "img")
	p.AddTargetBlankToFullyQualifiedLinks(true)
	p.RequireNoReferrerOnLinks(true)
	p.AllowAttrs("src").Matching(regexp.MustCompile(`^https?://\S+$`)).OnElements("img")
	p.AllowAttrs("alt").OnElements("img")
	p.AllowAttrs("width", "height").Matching(regexp.MustCompile(`^[0-9]{1,4}$`)).OnElements("img")
	p.AllowAttrs("colspan", "rowspan").Matching(regexp.MustCompile(`^[0-9]{1,2}$`)).OnElements("td", "th")
	p.AllowAttrs("class").Matching(regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*( [a-zA-Z][a-zA-Z0-9_-]*)*$`)).Globally()
	p.AllowStyles("text-align").MatchingEnum("left", "center", "right", "justify").Globally()
	return p
}
