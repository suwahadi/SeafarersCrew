// Package slug builds URL slugs following PRD rule 11.8.
package slug

import (
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// MaxLength is the longest slug allowed.
const MaxLength = 80

var specialLetters = map[rune]string{
	'ß': "ss", 'æ': "ae", 'Æ': "ae", 'ø': "o", 'Ø': "o", 'œ': "oe", 'Œ': "oe",
	'đ': "d", 'Đ': "d", 'ð': "d", 'Ð': "d", 'þ': "th", 'Þ': "th", 'ł': "l", 'Ł': "l",
	'ı': "i", '&': " and ",
}

// Make converts a title into an ASCII, lowercase, hyphenated slug of at most MaxLength characters.
func Make(title string) string {
	var b strings.Builder
	for _, r := range norm.NFKD.String(title) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if repl, ok := specialLetters[r]; ok {
			b.WriteString(repl)
			continue
		}
		b.WriteRune(r)
	}
	return collapse(strings.ToLower(b.String()), MaxLength)
}

// WithSuffix appends -n to base for n >= 2, trimming base so the result stays within MaxLength.
func WithSuffix(base string, n int) string {
	if n < 2 {
		return base
	}
	suffix := "-" + strconv.Itoa(n)
	return collapse(truncate(base, MaxLength-len(suffix)), MaxLength) + suffix
}

// Valid reports whether s is already a normalized slug.
func Valid(s string) bool {
	return s != "" && Make(s) == s
}

func collapse(s string, max int) string {
	var b strings.Builder
	pendingHyphen := false
	for _, r := range s {
		if r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			if pendingHyphen && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingHyphen = false
			b.WriteRune(r)
			continue
		}
		pendingHyphen = true
	}
	return strings.TrimRight(truncate(b.String(), max), "-")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return strings.TrimRight(s[:max], "-")
}
