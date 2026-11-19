package service

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
)

type cvMatch struct {
	Kind string    `json:"kind"`
	ID   uuid.UUID `json:"id"`
	By   string    `json:"by"`
}

type cvMatcher struct{ l repository.CVLookups }

var cvParenthesis = regexp.MustCompile(`\(([^)]*)\)`)

func newCVMatcher(l repository.CVLookups) cvMatcher { return cvMatcher{l: l} }

func normalizeCVText(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func cvNameVariants(s string) []string {
	full := normalizeCVText(s)
	short := s
	if i := strings.Index(short, "/"); i >= 0 {
		short = short[:i]
	}
	if i := strings.Index(short, "("); i >= 0 {
		short = short[:i]
	}
	out := []string{full}
	if v := normalizeCVText(short); v != "" && v != full {
		out = append(out, v)
	}
	return out
}

func cvAbbreviation(s string) string {
	if i := strings.LastIndex(s, "/"); i >= 0 {
		tail := s[i+1:]
		if j := strings.Index(tail, " - "); j >= 0 {
			tail = tail[:j]
		}
		if v := normalizeCVText(tail); v != "" {
			return v
		}
	}
	if m := cvParenthesis.FindStringSubmatch(s); m != nil {
		return normalizeCVText(m[1])
	}
	return ""
}

func cvFindName(items []repository.CVMasterItem, v string) (repository.CVMasterItem, bool) {
	for _, it := range items {
		for _, name := range cvNameVariants(it.Name) {
			if name == v {
				return it, true
			}
		}
	}
	return repository.CVMasterItem{}, false
}

func cvFindCode(items []repository.CVMasterItem, v string) (repository.CVMasterItem, bool) {
	for _, it := range items {
		if v != "" && (normalizeCVText(it.Code) == v || normalizeCVText(it.Name) == v) {
			return it, true
		}
	}
	return repository.CVMasterItem{}, false
}

type cvSource struct {
	kind  string
	items []repository.CVMasterItem
}

func (m cvMatcher) lookup(text string, sources []cvSource, useAbbreviation bool) (cvMatch, bool) {
	key := normalizeCVText(text)
	if key == "" {
		return cvMatch{}, false
	}
	for _, src := range sources {
		if id, ok := m.l.Aliases[src.kind][cvAliasKey(text)]; ok {
			return cvMatch{Kind: src.kind, ID: id, By: "alias"}, true
		}
	}
	for _, v := range cvNameVariants(text) {
		for _, src := range sources {
			if it, ok := cvFindName(src.items, v); ok {
				return cvMatch{Kind: src.kind, ID: it.ID, By: "name"}, true
			}
		}
	}
	if !useAbbreviation {
		return cvMatch{}, false
	}
	ab := cvAbbreviation(text)
	for _, src := range sources {
		if it, ok := cvFindCode(src.items, ab); ok {
			return cvMatch{Kind: src.kind, ID: it.ID, By: "code"}, true
		}
	}
	return cvMatch{}, false
}

func (m cvMatcher) certificate(text string) (cvMatch, bool) {
	return m.lookup(text, []cvSource{
		{repository.CVKindCertificateType, m.l.CertificateTypes}, {repository.CVKindDocumentType, m.l.DocumentTypes},
	}, true)
}

func (m cvMatcher) document(label string) (cvMatch, bool) {
	return m.lookup(label, []cvSource{{repository.CVKindDocumentType, m.l.DocumentTypes}}, true)
}

func (m cvMatcher) rank(text string) (cvMatch, bool) {
	key := normalizeCVText(text)
	if key == "" {
		return cvMatch{}, false
	}
	if id, ok := m.l.Aliases[repository.CVKindRank][cvAliasKey(text)]; ok {
		return cvMatch{Kind: repository.CVKindRank, ID: id, By: "alias"}, true
	}
	for _, it := range m.l.Ranks {
		if normalizeCVText(it.Code) == key {
			return cvMatch{Kind: repository.CVKindRank, ID: it.ID, By: "code"}, true
		}
	}
	for _, it := range m.l.Ranks {
		if normalizeCVText(it.Name) == key {
			return cvMatch{Kind: repository.CVKindRank, ID: it.ID, By: "name"}, true
		}
	}
	return cvMatch{}, false
}

func (m cvMatcher) vesselType(text string) (cvMatch, bool) {
	if mt, ok := m.lookup(text, []cvSource{{repository.CVKindVesselType, m.l.VesselTypes}}, false); ok {
		return mt, true
	}
	key := normalizeCVText(text)
	best, bestLen := cvMatch{}, 0
	for _, it := range m.l.VesselTypes {
		name := normalizeCVText(it.Name)
		if name != "" && strings.HasPrefix(key, name+" ") && len(name) > bestLen {
			best, bestLen = cvMatch{Kind: repository.CVKindVesselType, ID: it.ID, By: "prefix"}, len(name)
		}
	}
	return best, bestLen > 0
}

func (m cvMatcher) company(text string) *uuid.UUID {
	key := normalizeCVText(text)
	if key == "" {
		return nil
	}
	for _, it := range m.l.Companies {
		if normalizeCVText(it.Name) == key {
			id := it.ID
			return &id
		}
	}
	return nil
}

func (m cvMatcher) name(mt cvMatch) string {
	lists := map[string][]repository.CVMasterItem{
		repository.CVKindCertificateType: m.l.CertificateTypes, repository.CVKindDocumentType: m.l.DocumentTypes,
		repository.CVKindRank: m.l.Ranks, repository.CVKindVesselType: m.l.VesselTypes,
	}
	for _, it := range lists[mt.Kind] {
		if it.ID == mt.ID {
			return it.Name
		}
	}
	return ""
}

func cvIndonesian(nationality string) bool {
	switch normalizeCVText(nationality) {
	case "INDONESIAN", "INDONESIA", "WNI":
		return true
	}
	return false
}

func normalizeCVPhone(raw, nationality string) (string, bool) {
	s := strings.Map(func(r rune) rune {
		if strings.ContainsRune(" .-()/", r) {
			return -1
		}
		return r
	}, strings.TrimSpace(raw))
	if s == "" {
		return "", true
	}
	switch {
	case strings.HasPrefix(s, "00"):
		s = "+" + s[2:]
	case strings.HasPrefix(s, "0"):
		if !cvIndonesian(nationality) {
			return "", false
		}
		s = "+62" + s[1:]
	}
	errs := request.Errors{}
	errs.Phone("phone", s)
	if len(errs) > 0 {
		return "", false
	}
	return s, true
}

func mapCVRelationship(s string) string {
	switch normalizeCVText(s) {
	case "WIFE", "HUSBAND", "SPOUSE":
		return "spouse"
	case "FATHER", "MOTHER", "PARENT":
		return "parent"
	case "SON", "DAUGHTER", "CHILD":
		return "child"
	case "BROTHER", "SISTER", "SIBLING":
		return "sibling"
	}
	return "other"
}

func cvMaritalStatus(s string) string {
	switch normalizeCVText(s) {
	case "SINGLE":
		return "single"
	case "MARRIED":
		return "married"
	case "DIVORCED":
		return "divorced"
	case "WIDOW", "WIDOWED", "WIDOWER":
		return "widowed"
	}
	return ""
}
