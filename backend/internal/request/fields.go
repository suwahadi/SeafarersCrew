package request

import (
	"net/mail"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
)

var e164 = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)

// Errors collects field validation messages.
type Errors map[string]string

// Required checks a non-empty trimmed string no longer than max runes.
func (e Errors) Required(field, value string, max int) {
	if strings.TrimSpace(value) == "" {
		e[field] = "This field is required."
		return
	}
	e.MaxLen(field, value, max)
}

// MaxLen checks the rune length.
func (e Errors) MaxLen(field, value string, max int) {
	if utf8.RuneCountInString(value) > max {
		e.set(field, "This field may not exceed "+itoa(max)+" characters.")
	}
}

// Email checks an optional email.
func (e Errors) Email(field, value string) {
	if value == "" {
		return
	}
	if a, err := mail.ParseAddress(value); err != nil || a.Address != value || len(value) > 254 {
		e.set(field, "This must be a valid email address.")
	}
}

// Phone checks an optional E.164 phone number.
func (e Errors) Phone(field, value string) {
	if value != "" && !e164.MatchString(value) {
		e.set(field, "This must be in E.164 format, e.g. +628123456789.")
	}
}

// OneOf checks an optional value against allowed options.
func (e Errors) OneOf(field, value string, options ...string) {
	if value != "" && !slices.Contains(options, value) {
		e.set(field, "This must be one of: "+strings.Join(options, ", ")+".")
	}
}

// Date parses an optional YYYY-MM-DD value.
func (e Errors) Date(field, value string) *time.Time {
	if value == "" {
		return nil
	}
	t, err := datetime.ParseDate(value)
	if err != nil {
		e.set(field, "This must be a date in YYYY-MM-DD format.")
		return nil
	}
	return &t
}

// RequiredDate parses a mandatory YYYY-MM-DD value.
func (e Errors) RequiredDate(field, value string) time.Time {
	if value == "" {
		e.set(field, "This field is required.")
		return time.Time{}
	}
	t := e.Date(field, value)
	if t == nil {
		return time.Time{}
	}
	return *t
}

// UUID parses an optional UUID.
func (e Errors) UUID(field, value string) *uuid.UUID {
	if value == "" {
		return nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		e.set(field, "This must be a valid identifier.")
		return nil
	}
	return &id
}

// RequiredUUID parses a mandatory UUID.
func (e Errors) RequiredUUID(field, value string) uuid.UUID {
	if value == "" {
		e.set(field, "This field is required.")
		return uuid.Nil
	}
	id := e.UUID(field, value)
	if id == nil {
		return uuid.Nil
	}
	return *id
}

// Range checks an optional integer bound.
func (e Errors) Range(field string, value *int, min, max int) {
	if value != nil && (*value < min || *value > max) {
		e.set(field, "This must be between "+itoa(min)+" and "+itoa(max)+".")
	}
}

// After checks that end is on or after start when both exist.
func (e Errors) After(field string, start, end *time.Time, message string) {
	if start != nil && end != nil && end.Before(*start) {
		e.set(field, message)
	}
}

func (e Errors) set(field, msg string) {
	if _, exists := e[field]; !exists {
		e[field] = msg
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

// Trim trims surrounding whitespace from every string pointer.
func Trim(fields ...*string) {
	for _, f := range fields {
		*f = strings.TrimSpace(*f)
	}
}
