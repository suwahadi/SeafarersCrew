// Package link validates links authored in the CMS.
package link

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// MaxLength is the longest link accepted.
const MaxLength = 500

var phonePattern = regexp.MustCompile(`^\+?[0-9][0-9 ()-]{2,29}$`)

// Valid reports whether s is a site-relative path, an https URL, a mailto address or a tel number.
func Valid(s string) bool {
	if s == "" || len(s) > MaxLength || strings.IndexFunc(s, unicode.IsSpace) >= 0 || strings.IndexFunc(s, unicode.IsControl) >= 0 {
		return false
	}
	if strings.HasPrefix(s, "/") {
		return !strings.HasPrefix(s, "//") && !strings.Contains(s, `\`)
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return u.Host != ""
	case "mailto":
		return u.Opaque != "" && strings.Contains(u.Opaque, "@")
	case "tel":
		return phonePattern.MatchString(u.Opaque)
	}
	return false
}

// HTTPS reports whether s is an absolute https URL.
func HTTPS(s string) bool {
	if s == "" || len(s) > MaxLength || strings.IndexFunc(s, unicode.IsSpace) >= 0 {
		return false
	}
	u, err := url.Parse(s)
	return err == nil && strings.EqualFold(u.Scheme, "https") && u.Host != ""
}
