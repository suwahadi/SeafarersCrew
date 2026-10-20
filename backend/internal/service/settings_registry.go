package service

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FieldKind describes how a setting value is validated.
type FieldKind string

// Field kinds accepted by settings.
const (
	KindText     FieldKind = "text"
	KindTextarea FieldKind = "textarea"
	KindHTML     FieldKind = "html"
	KindEmail    FieldKind = "email"
	KindEmails   FieldKind = "emails"
	KindURL      FieldKind = "url"
	KindBool     FieldKind = "bool"
	KindInt      FieldKind = "int"
	KindColor    FieldKind = "color"
	KindSelect   FieldKind = "select"
	KindTime     FieldKind = "time"
	KindTimezone FieldKind = "timezone"
	KindFile     FieldKind = "file"
	KindList     FieldKind = "list"
	KindPhone    FieldKind = "phone"
	KindCoords   FieldKind = "coords"
)

// SettingField defines one key of a settings group.
type SettingField struct {
	Key     string    `json:"key"`
	Kind    FieldKind `json:"kind"`
	Secret  bool      `json:"secret"`
	Default any       `json:"default"`
	Options []string  `json:"options,omitempty"`
	Min     int       `json:"min,omitempty"`
	Max     int       `json:"max,omitempty"`
}

// SettingGroup is one tab of the Settings panel.
type SettingGroup struct {
	Name   string         `json:"name"`
	Fields []SettingField `json:"fields"`
}

// Field returns the field with key.
func (g SettingGroup) Field(key string) (SettingField, bool) {
	for _, f := range g.Fields {
		if f.Key == key {
			return f, true
		}
	}
	return SettingField{}, false
}

var homeSections = []string{"hero", "intro", "stats", "services", "news", "vacancies", "gallery", "testimonials", "clients", "cta"}

// SettingGroups lists every stored settings tab from PRD Section 9.
var SettingGroups = []SettingGroup{
	{Name: "general", Fields: []SettingField{
		{Key: "site_name", Kind: KindText, Default: "Seafarers Crew", Max: 120},
		{Key: "tagline", Kind: KindText, Default: "Trusted offshore crew manning agency", Max: 200},
		{Key: "admin_panel_title", Kind: KindText, Default: "Seafarers Crew Admin", Max: 120},
		{Key: "default_language", Kind: KindText, Default: "en", Max: 10},
		{Key: "timezone", Kind: KindTimezone, Default: "Asia/Jakarta"},
		{Key: "date_format", Kind: KindSelect, Default: "DD-MMM-YYYY", Options: []string{"DD-MMM-YYYY", "DD/MM/YYYY", "YYYY-MM-DD", "MM/DD/YYYY"}},
		{Key: "maintenance_mode", Kind: KindBool, Default: false},
	}},
	{Name: "company", Fields: []SettingField{
		{Key: "legal_name", Kind: KindText, Default: "PT Seafarers Crew Indonesia", Max: 200},
		{Key: "logo_light_file_id", Kind: KindFile, Default: ""},
		{Key: "logo_dark_file_id", Kind: KindFile, Default: ""},
		{Key: "favicon_file_id", Kind: KindFile, Default: ""},
		{Key: "address", Kind: KindTextarea, Default: "Jl. Example No. 1, Jakarta 10110", Max: 500},
		{Key: "country", Kind: KindText, Default: "Indonesia", Max: 80},
		{Key: "phone", Kind: KindPhone, Default: "+62210000000"},
		{Key: "email", Kind: KindEmail, Default: "info@example.com"},
		{Key: "website", Kind: KindURL, Default: "https://example.com"},
		{Key: "tax_number", Kind: KindText, Default: "", Max: 40},
		{Key: "signatory_name", Kind: KindText, Default: "Director", Max: 120},
		{Key: "signatory_title", Kind: KindText, Default: "Director", Max: 120},
	}},
	{Name: "contact", Fields: []SettingField{
		{Key: "office_address", Kind: KindTextarea, Default: "Jl. Example No. 1, Jakarta 10110", Max: 500},
		{Key: "map_coordinates", Kind: KindCoords, Default: "-6.200000,106.816666"},
		{Key: "phones", Kind: KindList, Default: []string{"+62210000000"}, Max: 5},
		{Key: "public_email", Kind: KindEmail, Default: "info@example.com"},
		{Key: "whatsapp_number", Kind: KindPhone, Default: ""},
		{Key: "whatsapp_message", Kind: KindText, Default: "Hello, I would like to ask about crew vacancies.", Max: 300},
		{Key: "linkedin_url", Kind: KindURL, Default: ""},
		{Key: "facebook_url", Kind: KindURL, Default: ""},
		{Key: "instagram_url", Kind: KindURL, Default: ""},
		{Key: "youtube_url", Kind: KindURL, Default: ""},
		{Key: "x_url", Kind: KindURL, Default: ""},
	}},
	{Name: "branding", Fields: []SettingField{
		{Key: "primary_color", Kind: KindColor, Default: "#0B2545"},
		{Key: "accent_color", Kind: KindColor, Default: "#13A89E"},
		{Key: "footer_text", Kind: KindTextarea, Default: "Connecting skilled seafarers with the offshore industry.", Max: 500},
	}},
	{Name: "seo", Fields: []SettingField{
		{Key: "meta_title_pattern", Kind: KindText, Default: "%s | Seafarers Crew", Max: 120},
		{Key: "meta_description", Kind: KindTextarea, Default: "Offshore crew manning agency providing qualified, STCW-compliant seafarers.", Max: 300},
		{Key: "og_image_file_id", Kind: KindFile, Default: ""},
		{Key: "google_site_verification", Kind: KindText, Default: "", Max: 120},
		{Key: "robots_indexing", Kind: KindBool, Default: true},
	}},
	{Name: "email", Fields: []SettingField{
		{Key: "brevo_api_key", Kind: KindText, Secret: true, Default: ""},
		{Key: "sender_name", Kind: KindText, Default: "Seafarers Crew", Max: 120},
		{Key: "sender_email", Kind: KindEmail, Default: "no-reply@example.com"},
		{Key: "admin_notification_emails", Kind: KindEmails, Default: []string{"admin@example.com"}, Max: 10},
		{Key: "reply_to", Kind: KindEmail, Default: ""},
	}},
	{Name: "storage", Fields: []SettingField{
		{Key: "provider", Kind: KindSelect, Default: "s3_compatible", Options: []string{"aws_s3", "s3_compatible"}},
		{Key: "endpoint", Kind: KindURL, Default: ""},
		{Key: "region", Kind: KindText, Default: "us-east-1", Max: 40},
		{Key: "bucket", Kind: KindText, Default: "", Max: 63},
		{Key: "access_key_id", Kind: KindText, Default: "", Max: 128},
		{Key: "secret_access_key", Kind: KindText, Secret: true, Default: ""},
		{Key: "use_path_style", Kind: KindBool, Default: true},
		{Key: "presign_minutes", Kind: KindInt, Default: 5, Min: 1, Max: 60},
	}},
	{Name: "uploads", Fields: []SettingField{
		{Key: "max_file_size_mb", Kind: KindInt, Default: 10, Min: 1, Max: 100},
		{Key: "max_applicant_files", Kind: KindInt, Default: 10, Min: 1, Max: 20},
		{Key: "webp_quality", Kind: KindInt, Default: 80, Min: 40, Max: 100},
		{Key: "max_image_dimension", Kind: KindInt, Default: 2000, Min: 480, Max: 8000},
		{Key: "pdf_preset", Kind: KindSelect, Default: "ebook", Options: []string{"screen", "ebook", "printer", "prepress"}},
	}},
	{Name: "security", Fields: []SettingField{
		{Key: "turnstile_site_key", Kind: KindText, Default: "", Max: 120},
		{Key: "turnstile_secret_key", Kind: KindText, Secret: true, Default: ""},
	}},
	{Name: "backup", Fields: []SettingField{
		{Key: "enabled", Kind: KindBool, Default: true},
		{Key: "run_time", Kind: KindTime, Default: "02:00"},
		{Key: "retention_days", Kind: KindInt, Default: 30, Min: 1, Max: 3650},
		{Key: "prefix", Kind: KindText, Default: "backups/", Max: 120},
	}},
	{Name: "alerts", Fields: []SettingField{
		{Key: "expiry_threshold_1", Kind: KindInt, Default: 30, Min: 1, Max: 365},
		{Key: "expiry_threshold_2", Kind: KindInt, Default: 60, Min: 1, Max: 365},
		{Key: "expiry_threshold_3", Kind: KindInt, Default: 90, Min: 1, Max: 365},
	}},
	{Name: "website", Fields: []SettingField{
		{Key: "hero_autoplay_seconds", Kind: KindInt, Default: 6, Min: 2, Max: 30},
		{Key: "home_sections", Kind: KindList, Default: homeSections, Options: homeSections, Max: len(homeSections)},
		{Key: "news_per_page", Kind: KindInt, Default: 9, Min: 3, Max: 48},
		{Key: "career_intro_html", Kind: KindHTML, Default: "<p>Join our fleet of professional seafarers. Browse open positions below and apply online.</p>", Max: 5000},
		{Key: "analytics_enabled", Kind: KindBool, Default: false},
	}},
}

// FindSettingGroup returns the group with name.
func FindSettingGroup(name string) (SettingGroup, bool) {
	for _, g := range SettingGroups {
		if g.Name == name {
			return g, true
		}
	}
	return SettingGroup{}, false
}

var (
	colorPattern  = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	timePattern   = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
	phonePattern  = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)
	coordsPattern = regexp.MustCompile(`^-?([0-8]?[0-9](\.[0-9]+)?|90(\.0+)?),\s*-?((1[0-7][0-9]|[0-9]?[0-9])(\.[0-9]+)?|180(\.0+)?)$`)
)

// Normalize validates raw against the field and returns the canonical value.
func (f SettingField) Normalize(raw json.RawMessage) (any, error) {
	switch f.Kind {
	case KindBool:
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, fmt.Errorf("must be true or false")
		}
		return b, nil
	case KindInt:
		return f.normalizeInt(raw)
	case KindList, KindEmails:
		return f.normalizeList(raw)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("must be a string")
	}
	s = strings.TrimSpace(s)
	if f.Max > 0 && len([]rune(s)) > f.Max {
		return nil, fmt.Errorf("may not exceed %d characters", f.Max)
	}
	if s == "" {
		return s, nil
	}
	return s, f.validateString(s)
}

func (f SettingField) normalizeInt(raw json.RawMessage) (any, error) {
	var n float64
	if err := json.Unmarshal(raw, &n); err != nil || n != float64(int(n)) {
		return nil, fmt.Errorf("must be a whole number")
	}
	if int(n) < f.Min || int(n) > f.Max {
		return nil, fmt.Errorf("must be between %d and %d", f.Min, f.Max)
	}
	return int(n), nil
}

func (f SettingField) normalizeList(raw json.RawMessage) (any, error) {
	var items []string
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("must be a list of strings")
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		it = strings.TrimSpace(it)
		if it == "" || slices.Contains(out, it) {
			continue
		}
		if f.Kind == KindEmails {
			if _, err := mail.ParseAddress(it); err != nil {
				return nil, fmt.Errorf("contains an invalid email: %s", it)
			}
		}
		if len(f.Options) > 0 && !slices.Contains(f.Options, it) {
			return nil, fmt.Errorf("contains an unknown value: %s", it)
		}
		out = append(out, it)
	}
	if f.Max > 0 && len(out) > f.Max {
		return nil, fmt.Errorf("may contain at most %d items", f.Max)
	}
	return out, nil
}

func (f SettingField) validateString(s string) error {
	switch f.Kind {
	case KindEmail:
		if a, err := mail.ParseAddress(s); err != nil || a.Address != s {
			return fmt.Errorf("must be a valid email address")
		}
	case KindURL:
		u, err := url.Parse(s)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("must be a valid http or https URL")
		}
	case KindColor:
		if !colorPattern.MatchString(s) {
			return fmt.Errorf("must be a hex color such as #0B2545")
		}
	case KindSelect:
		if !slices.Contains(f.Options, s) {
			return fmt.Errorf("must be one of %s", strings.Join(f.Options, ", "))
		}
	case KindTime:
		if !timePattern.MatchString(s) {
			return fmt.Errorf("must be a time in HH:MM format")
		}
	case KindTimezone:
		if _, err := time.LoadLocation(s); err != nil {
			return fmt.Errorf("must be a valid IANA timezone")
		}
	case KindFile:
		if _, err := uuid.Parse(s); err != nil {
			return fmt.Errorf("must be a file ID")
		}
	case KindPhone:
		if !phonePattern.MatchString(s) {
			return fmt.Errorf("must be in E.164 format such as +628123456789")
		}
	case KindCoords:
		if !coordsPattern.MatchString(s) {
			return fmt.Errorf("must be latitude,longitude")
		}
	}
	return nil
}
