package resource

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

type testURLs struct{}

func (testURLs) URL(key string) string {
	if rest, ok := strings.CutPrefix(key, "public/"); ok {
		return "https://api.example.com/media/" + rest
	}
	return ""
}

func img(id uuid.UUID, key string) model.Image {
	return model.Image{ID: id, StorageKey: key, MimeType: "image/webp", Width: 1600, Height: 900,
		Variants: []model.FileVariant{{Width: 480, Height: 270, StorageKey: "public/x-480.webp"}, {Width: 960, Height: 540, StorageKey: "public/x-960.webp"}}}
}

func toJSON(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

func TestPublicSiteResolvesMenusAndSkipsUnpublishedPages(t *testing.T) {
	pageID := uuid.New()
	home := model.SystemHome
	published := &model.MenuPage{Title: "About", Slug: "about-us", Status: model.StatusPublished}
	site := NewPublicSite(model.SiteData{
		Settings: map[string]map[string]any{},
		Menus: model.Menus{Header: []model.MenuItem{
			{Label: "Home", PageID: &pageID, Page: &model.MenuPage{Slug: "home", SystemKey: &home, Status: model.StatusPublished}},
			{Label: "About", PageID: &pageID, Page: published, Children: []model.MenuItem{
				{Label: "Draft", PageID: &pageID, Page: &model.MenuPage{Slug: "draft", Status: model.StatusDraft}},
				{Label: "Safety", URL: "/safety"},
			}},
			{Label: "Draft top", PageID: &pageID, Page: &model.MenuPage{Slug: "wip", Status: model.StatusDraft}, Children: []model.MenuItem{{Label: "Hidden child", URL: "/x"}}},
			{Label: "Partner", URL: "https://partner.example.com", OpenInNewTab: true},
		}},
	}, testURLs{})

	require.Len(t, site.Menus.Header, 3)
	assert.Equal(t, "/", site.Menus.Header[0].Href, "system pages use their fixed paths")
	assert.Equal(t, "/about-us", site.Menus.Header[1].Href)
	require.Len(t, site.Menus.Header[1].Children, 1)
	assert.Equal(t, "/safety", site.Menus.Header[1].Children[0].Href)
	assert.True(t, site.Menus.Header[2].OpenInNewTab)
	assert.NotNil(t, site.Menus.Footer)
}

func TestPublicSiteMapsContactBrandingAndPublicKeysOnly(t *testing.T) {
	logo := uuid.New()
	site := NewPublicSite(model.SiteData{
		Settings: map[string]map[string]any{
			"general":  {"site_name": "Seafarers Crew", "tagline": "Crew", "maintenance_mode": true},
			"company":  {"logo_light_file_id": logo.String(), "logo_dark_file_id": uuid.NewString()},
			"contact":  {"map_coordinates": "-6.2,106.81", "phones": []string{"+62210000000"}, "whatsapp_number": "+628123456789", "whatsapp_message": "Hello & welcome", "linkedin_url": "https://linkedin.com/x"},
			"branding": {"primary_color": "#0B2545"},
			"seo":      {"robots_indexing": true},
			"website":  {"hero_autoplay_seconds": 7, "analytics_enabled": true},
			"security": {"turnstile_site_key": "0xKEY", "turnstile_secret_key": "SECRET"},
			"email":    {"brevo_api_key": "SECRET"},
		},
		Images: map[uuid.UUID]model.Image{logo: img(logo, "public/logo/2026/10/a.svg")},
	}, testURLs{})

	assert.True(t, site.MaintenanceMode)
	require.NotNil(t, site.Branding.LogoLight)
	assert.Equal(t, "https://api.example.com/media/logo/2026/10/a.svg", site.Branding.LogoLight.URL)
	assert.Nil(t, site.Branding.LogoDark, "an unknown file is simply omitted")
	require.NotNil(t, site.Contact.Coordinates)
	assert.InDelta(t, -6.2, site.Contact.Coordinates.Lat, 1e-9)
	require.NotNil(t, site.Contact.WhatsApp)
	assert.Equal(t, "https://wa.me/628123456789?text=Hello+%26+welcome", site.Contact.WhatsApp.URL)
	assert.Equal(t, 7, site.HeroAutoplaySeconds)
	assert.Equal(t, "0xKEY", site.TurnstileSiteKey)

	raw, err := json.Marshal(site)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "SECRET")
}

func TestPublicSiteOmitsWhatsAppAndCoordinatesWhenNotConfigured(t *testing.T) {
	site := NewPublicSite(model.SiteData{Settings: map[string]map[string]any{"contact": {"map_coordinates": "nonsense"}}}, testURLs{})
	assert.Nil(t, site.Contact.WhatsApp)
	assert.Nil(t, site.Contact.Coordinates)
	assert.Equal(t, []string{}, site.Contact.Phones)
}

func TestPublicPageResolvesBlocksForRendering(t *testing.T) {
	imageID, galleryID, missingGallery := uuid.New(), uuid.New(), uuid.New()
	cardImage := uuid.New()
	data := model.PageData{
		Page: model.Page{
			Title: "About", Slug: "about-us", Status: model.StatusPublished, UpdatedAt: time.Now(),
			Blocks: []model.Block{
				{ID: "1", Type: model.BlockRichText, Settings: &model.RichTextSettings{HTML: "<p>x</p>"}},
				{ID: "2", Type: model.BlockImageText, Settings: &model.ImageTextSettings{ImageFileID: &imageID, Heading: "Who we are", ImagePosition: "right"}},
				{ID: "3", Type: model.BlockGallery, Settings: &model.GallerySettings{ImageIDs: []uuid.UUID{galleryID, missingGallery}, Columns: 3, Lightbox: true}},
				{ID: "4", Type: model.BlockVideo, Settings: &model.VideoSettings{URL: "https://youtu.be/dQw4w9WgXcQ", Title: "Intro", Provider: "youtube", VideoID: "dQw4w9WgXcQ"}},
				{ID: "5", Type: model.BlockVideo, Settings: &model.VideoSettings{URL: "https://vimeo.com/76979871", Provider: "vimeo", VideoID: "76979871"}},
				{ID: "6", Type: model.BlockCTA, Settings: &model.CTASettings{Heading: "Join", BackgroundColor: "#0B2545"}},
				{ID: "7", Type: model.BlockStats, Settings: &model.StatsSettings{Items: []model.StatsItemSetting{{Number: 5, Suffix: "+", Label: "Ships"}}}},
				{ID: "8", Type: model.BlockFAQ, Settings: &model.FAQSettings{Items: []model.FAQItem{{Question: "Fees?", AnswerHTML: "<p>None</p>"}}}},
				{ID: "9", Type: model.BlockCardGrid, Settings: &model.CardGridSettings{Columns: 3, Cards: []model.Card{{Title: "Crewing", ImageFileID: &cardImage, Link: "/services"}}}},
				{ID: "10", Type: model.BlockGallery, Settings: &model.GallerySettings{ImageIDs: []uuid.UUID{missingGallery}, Columns: 2}},
			},
		},
		Images: map[uuid.UUID]model.Image{imageID: img(imageID, "public/cms_image/2026/10/a.webp"), cardImage: img(cardImage, "public/cms_image/2026/10/c.webp")},
		Gallery: map[uuid.UUID]model.GalleryImage{galleryID: {
			MasterBase: model.MasterBase{ID: galleryID}, Caption: "On deck", AltText: "Crew on deck", Image: ptrImage(img(uuid.New(), "public/cms_image/2026/10/g.webp")),
		}},
	}
	page := NewPublicPage(data, testURLs{})

	require.Len(t, page.Blocks, 9, "a gallery block without any live image is dropped")
	settings := func(i int) map[string]any { return toJSON(t, page.Blocks[i].Settings) }

	assert.Equal(t, "<p>x</p>", settings(0)["html"])
	imageText := settings(1)
	assert.Equal(t, "right", imageText["image_position"])
	image := imageText["image"].(map[string]any)
	assert.Equal(t, "https://api.example.com/media/cms_image/2026/10/a.webp", image["url"])
	assert.Equal(t, "Who we are", image["alt"])
	assert.NotEmpty(t, image["sources"])

	gallery := settings(2)
	images := gallery["images"].([]any)
	require.Len(t, images, 1, "gallery images that are gone are skipped")
	first := images[0].(map[string]any)
	assert.Equal(t, "On deck", first["caption"])
	assert.Equal(t, "Crew on deck", first["image"].(map[string]any)["alt"])
	assert.Equal(t, true, gallery["lightbox"])

	youtube := settings(3)
	assert.Equal(t, "youtube", youtube["provider"])
	assert.Equal(t, "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ", youtube["embed_url"])
	assert.Equal(t, "https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg", youtube["thumbnail_url"])
	vimeo := settings(4)
	assert.Equal(t, "https://player.vimeo.com/video/76979871", vimeo["embed_url"])
	assert.Equal(t, "", vimeo["thumbnail_url"])

	assert.Nil(t, settings(5)["background_image"])
	assert.Equal(t, "#0B2545", settings(5)["background_color"])
	assert.Len(t, settings(6)["items"], 1)
	assert.Len(t, settings(7)["items"], 1)
	cards := settings(8)["cards"].([]any)
	assert.Equal(t, "Crewing", cards[0].(map[string]any)["image"].(map[string]any)["alt"])
}

func ptrImage(i model.Image) *model.Image { return &i }

func TestPublicPageForASystemPageHasNoBlocksAndAFixedPath(t *testing.T) {
	key := model.SystemContact
	page := NewPublicPage(model.PageData{Page: model.Page{Title: "Contact", Slug: "contact", SystemKey: &key, IntroHTML: "<p>Hi</p>"}}, testURLs{})
	assert.Equal(t, "/contact", page.Path)
	assert.Equal(t, []PublicBlock{}, page.Blocks)
	assert.Equal(t, "<p>Hi</p>", page.IntroHTML)
}

func TestPublicHomeMapsSectionsAndKeepsEmptyListsAsArrays(t *testing.T) {
	closing := time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)
	vesselType := uuid.New()
	home := NewPublicHome(model.HomeData{
		Page:      &model.Page{Title: "Home", IntroHTML: "<p>Welcome</p>"},
		Sections:  []string{"hero", "news"},
		Autoplay:  6,
		Vacancies: []model.Vacancy{{PublicID: "VACAAAAA1", Title: "Master", Slug: "master", RankName: "Master", VesselTypeID: &vesselType, VesselTypeName: "AHTS", Location: "Batam", ClosingDate: &closing}, {Slug: "no-close"}},
	}, testURLs{})

	body := toJSON(t, home)
	assert.Equal(t, []any{"hero", "news"}, body["sections"])
	assert.Equal(t, "Home", body["page"].(map[string]any)["title"])
	for _, key := range []string{"stats", "news", "gallery", "testimonials", "client_logos"} {
		assert.Equal(t, []any{}, body[key], key)
	}
	assert.Equal(t, []any{}, body["hero"].(map[string]any)["slides"])
	vacancies := body["vacancies"].([]any)
	first := vacancies[0].(map[string]any)
	assert.Equal(t, "2026-11-05", first["closing_date"])
	assert.Equal(t, "VACAAAAA1", first["id"])
	assert.Equal(t, "AHTS", first["vessel_type"].(map[string]any)["name"])
	assert.Equal(t, "Master", first["rank"].(map[string]any)["name"])
	assert.Nil(t, vacancies[1].(map[string]any)["closing_date"])
}

func TestPublicNewsArticleCarriesRelatedCards(t *testing.T) {
	at := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	article := NewPublicNewsArticle(model.NewsDetailData{
		Article: model.News{Title: "Crew change", Slug: "crew-change", BodyHTML: "<p>Body</p>", PublishedAt: &at},
		Related: []model.News{{Title: "Other", Slug: "other", PublishedAt: &at}},
	}, testURLs{})
	body := toJSON(t, article)
	assert.Equal(t, "<p>Body</p>", body["body_html"])
	assert.Len(t, body["related"], 1)
	assert.Nil(t, body["thumbnail"])
	assert.Equal(t, "2026-10-01T08:00:00Z", body["published_at"])
}

func TestAdminImageUsesTheSmallestVariantAsThumbnail(t *testing.T) {
	id := uuid.New()
	image := img(id, "public/cms_image/2026/10/a.webp")
	got := NewAdminImage(&image, testURLs{})
	require.NotNil(t, got)
	assert.Equal(t, "https://api.example.com/media/x-480.webp", got.ThumbURL)
	assert.Equal(t, "https://api.example.com/media/cms_image/2026/10/a.webp", got.URL)
	assert.Equal(t, id, got.FileID)
	assert.Nil(t, NewAdminImage(nil, testURLs{}))
	private := model.Image{StorageKey: "crew/private.webp"}
	assert.Nil(t, NewAdminImage(&private, testURLs{}), "private files have no public URL")
}

func TestPublicSitemapKeepsLastModifiedTimes(t *testing.T) {
	at := time.Date(2026, 10, 1, 8, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	out := toJSON(t, NewPublicSitemap(model.SitemapData{Pages: []model.SitemapEntry{{Path: "/about", LastMod: at}}}))
	page := out["pages"].([]any)[0].(map[string]any)
	assert.Equal(t, "2026-10-01T01:00:00Z", page["lastmod"])
	assert.Equal(t, []any{}, out["vacancies"])
}
