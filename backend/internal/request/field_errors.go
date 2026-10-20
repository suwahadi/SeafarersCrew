package request

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// FieldErrors collects per-field validation messages.
type FieldErrors map[string]string

func (f FieldErrors) set(field, msg string) {
	if _, exists := f[field]; !exists {
		f[field] = msg
	}
}

// Required checks a trimmed string is present and within max runes.
func (f FieldErrors) Required(field, value string, max int) {
	if value == "" {
		f.set(field, "The "+label(field)+" is required.")
		return
	}
	f.MaxLen(field, value, max)
}

// MaxLen checks a string is within max runes.
func (f FieldErrors) MaxLen(field, value string, max int) {
	if utf8.RuneCountInString(value) > max {
		f.set(field, fmt.Sprintf("The %s may not exceed %d characters.", label(field), max))
	}
}

// OneOf checks value is one of options.
func (f FieldErrors) OneOf(field, value string, options ...string) {
	if !slices.Contains(options, value) {
		f.set(field, "The "+label(field)+" must be one of: "+strings.Join(options, ", ")+".")
	}
}

// Matches checks value against a pattern when value is not empty.
func (f FieldErrors) Matches(field, value string, pattern *regexp.Regexp, msg string) {
	if value != "" && !pattern.MatchString(value) {
		f.set(field, msg)
	}
}

// Email checks an optional email address.
func (f FieldErrors) Email(field, value string) {
	if value == "" {
		return
	}
	if a, err := mail.ParseAddress(value); err != nil || a.Address != value || len(value) > 254 {
		f.set(field, "The "+label(field)+" must be a valid email address.")
	}
}

// URL checks an optional http or https URL.
func (f FieldErrors) URL(field, value string) {
	if value == "" {
		return
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(value) > 500 {
		f.set(field, "The "+label(field)+" must be a valid http or https URL.")
	}
}

// Range checks an optional integer lies within [min, max].
func (f FieldErrors) Range(field string, value *int, min, max int) {
	if value != nil && (*value < min || *value > max) {
		f.set(field, fmt.Sprintf("The %s must be between %d and %d.", label(field), min, max))
	}
}

func label(field string) string {
	return strings.ReplaceAll(strings.TrimSuffix(field, "_id"), "_", " ")
}

func boolOr(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}
