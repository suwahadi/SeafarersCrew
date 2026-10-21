package resource

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

// PublicBranding carries logos, favicon, colors and footer text.
type PublicBranding struct {
	LogoLight    *PublicImage `json:"logo_light"`
	LogoDark     *PublicImage `json:"logo_dark"`
	Favicon      *PublicImage `json:"favicon"`
	PrimaryColor string       `json:"primary_color"`
	AccentColor  string       `json:"accent_color"`
	FooterText   string       `json:"footer_text"`
}

// PublicCoordinates is a map position.
type PublicCoordinates struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// PublicWhatsApp is the WhatsApp contact.
type PublicWhatsApp struct {
	Number  string `json:"number"`
	Message string `json:"message"`
	URL     string `json:"url"`
}

// PublicContact is the office contact block.
type PublicContact struct {
	OfficeAddress string             `json:"office_address"`
	Coordinates   *PublicCoordinates `json:"coordinates"`
	Phones        []string           `json:"phones"`
	Email         string             `json:"email"`
	WhatsApp      *PublicWhatsApp    `json:"whatsapp"`
}

// PublicSocial lists the configured social profiles.
type PublicSocial struct {
	LinkedIn  string `json:"linkedin"`
	Facebook  string `json:"facebook"`
	Instagram string `json:"instagram"`
	YouTube   string `json:"youtube"`
	X         string `json:"x"`
}

// PublicOrganization is the company identity for structured data.
type PublicOrganization struct {
	LegalName string `json:"legal_name"`
	Website   string `json:"website"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Country   string `json:"country"`
}

// PublicSEO holds the site-wide SEO defaults.
type PublicSEO struct {
	MetaTitlePattern       string       `json:"meta_title_pattern"`
	MetaDescription        string       `json:"meta_description"`
	DefaultOGImage         *PublicImage `json:"default_og_image"`
	GoogleSiteVerification string       `json:"google_site_verification"`
	RobotsIndexing         bool         `json:"robots_indexing"`
}

// PublicMenuItem is a resolved menu link.
type PublicMenuItem struct {
	Label        string           `json:"label"`
	Href         string           `json:"href"`
	OpenInNewTab bool             `json:"open_in_new_tab"`
	Children     []PublicMenuItem `json:"children"`
}

// PublicMenus holds both resolved menus.
type PublicMenus struct {
	Header []PublicMenuItem `json:"header"`
	Footer []PublicMenuItem `json:"footer"`
}

// PublicLanguage is one language switcher entry.
type PublicLanguage struct {
	Name     string       `json:"name"`
	Code     string       `json:"code"`
	FlagCode string       `json:"flag_code"`
	Flag     *PublicImage `json:"flag"`
	IsSource bool         `json:"is_source"`
}

// PublicSite is the payload of GET /site.
type PublicSite struct {
	Name                string             `json:"name"`
	Tagline             string             `json:"tagline"`
	DefaultLanguage     string             `json:"default_language"`
	MaintenanceMode     bool               `json:"maintenance_mode"`
	Organization        PublicOrganization `json:"organization"`
	Branding            PublicBranding     `json:"branding"`
	Contact             PublicContact      `json:"contact"`
	Social              PublicSocial       `json:"social"`
	SEO                 PublicSEO          `json:"seo"`
	Menus               PublicMenus        `json:"menus"`
	Languages           []PublicLanguage   `json:"languages"`
	HeroAutoplaySeconds int                `json:"hero_autoplay_seconds"`
	AnalyticsEnabled    bool               `json:"analytics_enabled"`
	TurnstileSiteKey    string             `json:"turnstile_site_key"`
}

// NewPublicSite maps the site shell data; secrets are never part of SiteData.
func NewPublicSite(d model.SiteData, urls MediaURLer) PublicSite {
	general, company, contact := d.Settings["general"], d.Settings["company"], d.Settings["contact"]
	branding, seo, website, security := d.Settings["branding"], d.Settings["seo"], d.Settings["website"], d.Settings["security"]
	image := func(values map[string]any, key, alt string) *PublicImage {
		id, ok := service.SettingFileID(values, key)
		if !ok {
			return nil
		}
		img, found := d.Images[id]
		if !found {
			return nil
		}
		return NewPublicImage(&img, urls, alt)
	}
	name := str(general, "site_name")
	return PublicSite{
		Name:            name,
		Tagline:         str(general, "tagline"),
		DefaultLanguage: str(general, "default_language"),
		MaintenanceMode: flag(general, "maintenance_mode"),
		Organization: PublicOrganization{
			LegalName: str(company, "legal_name"), Website: str(company, "website"), Email: str(company, "email"),
			Phone: str(company, "phone"), Address: str(company, "address"), Country: str(company, "country"),
		},
		Branding: PublicBranding{
			LogoLight: image(company, "logo_light_file_id", name), LogoDark: image(company, "logo_dark_file_id", name),
			Favicon: image(company, "favicon_file_id", name), PrimaryColor: str(branding, "primary_color"),
			AccentColor: str(branding, "accent_color"), FooterText: str(branding, "footer_text"),
		},
		Contact: PublicContact{
			OfficeAddress: str(contact, "office_address"), Coordinates: parseCoordinates(str(contact, "map_coordinates")),
			Phones: strList(contact, "phones"), Email: str(contact, "public_email"),
			WhatsApp: newWhatsApp(str(contact, "whatsapp_number"), str(contact, "whatsapp_message")),
		},
		Social: PublicSocial{
			LinkedIn: str(contact, "linkedin_url"), Facebook: str(contact, "facebook_url"), Instagram: str(contact, "instagram_url"),
			YouTube: str(contact, "youtube_url"), X: str(contact, "x_url"),
		},
		SEO: PublicSEO{
			MetaTitlePattern: str(seo, "meta_title_pattern"), MetaDescription: str(seo, "meta_description"),
			DefaultOGImage: image(seo, "og_image_file_id", name), GoogleSiteVerification: str(seo, "google_site_verification"),
			RobotsIndexing: flag(seo, "robots_indexing"),
		},
		Menus: PublicMenus{Header: newPublicMenu(d.Menus.Header), Footer: newPublicMenu(d.Menus.Footer)},
		Languages: MapAll(d.Languages, func(l model.Language) PublicLanguage {
			return PublicLanguage{Name: l.Name, Code: l.Code, FlagCode: l.FlagCode, Flag: NewPublicImage(l.Flag, urls, l.Name), IsSource: l.IsSource}
		}),
		HeroAutoplaySeconds: intOr(website, "hero_autoplay_seconds", 6),
		AnalyticsEnabled:    flag(website, "analytics_enabled"),
		TurnstileSiteKey:    str(security, "turnstile_site_key"),
	}
}

func newPublicMenu(items []model.MenuItem) []PublicMenuItem {
	out := make([]PublicMenuItem, 0, len(items))
	for _, it := range items {
		href, ok := menuHref(it)
		if !ok {
			continue
		}
		out = append(out, PublicMenuItem{Label: it.Label, Href: href, OpenInNewTab: it.OpenInNewTab, Children: newPublicMenu(it.Children)})
	}
	return out
}

func menuHref(it model.MenuItem) (string, bool) {
	if it.PageID == nil {
		return it.URL, it.URL != ""
	}
	if it.Page == nil || it.Page.Status != model.StatusPublished {
		return "", false
	}
	if it.Page.SystemKey != nil {
		return model.SystemPath(*it.Page.SystemKey), true
	}
	return "/" + it.Page.Slug, true
}

func newWhatsApp(number, message string) *PublicWhatsApp {
	digits := strings.TrimPrefix(number, "+")
	if digits == "" {
		return nil
	}
	link := "https://wa.me/" + digits
	if message != "" {
		link += "?text=" + url.QueryEscape(message)
	}
	return &PublicWhatsApp{Number: number, Message: message, URL: link}
}

func parseCoordinates(s string) *PublicCoordinates {
	lat, lng, ok := strings.Cut(s, ",")
	if !ok {
		return nil
	}
	la, err1 := strconv.ParseFloat(strings.TrimSpace(lat), 64)
	ln, err2 := strconv.ParseFloat(strings.TrimSpace(lng), 64)
	if err1 != nil || err2 != nil {
		return nil
	}
	return &PublicCoordinates{Lat: la, Lng: ln}
}

func str(values map[string]any, key string) string {
	s, _ := values[key].(string)
	return s
}

func flag(values map[string]any, key string) bool {
	b, _ := values[key].(bool)
	return b
}

func intOr(values map[string]any, key string, fallback int) int {
	if n, ok := values[key].(int); ok && n > 0 {
		return n
	}
	return fallback
}

func strList(values map[string]any, key string) []string {
	list, _ := values[key].([]string)
	if list == nil {
		return []string{}
	}
	return list
}
