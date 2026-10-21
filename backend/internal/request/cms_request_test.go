package request

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decodeInto(t *testing.T, dst Validator, body string) map[string]string {
	t.Helper()
	require.NoError(t, json.Unmarshal([]byte(body), dst))
	return dst.Validate()
}

func TestCMSRequestsRejectInvalidFields(t *testing.T) {
	id := uuid.NewString()
	cases := []struct {
		name string
		dst  Validator
		body string
		keys []string
	}{
		{"hero ok", &HeroSlide{}, `{"image_file_id":"` + id + `","heading":"Crew","button_label":"Apply","button_link":"/career"}`, nil},
		{"hero needs image", &HeroSlide{}, `{"heading":"Crew"}`, []string{"image_file_id"}},
		{"hero link", &HeroSlide{}, `{"image_file_id":"` + id + `","button_label":"Go","button_link":"ftp://x"}`, []string{"button_link"}},
		{"hero label without link", &HeroSlide{}, `{"image_file_id":"` + id + `","button_label":"Go"}`, []string{"button_link"}},
		{"gallery ok", &GalleryImage{}, `{"image_file_id":"` + id + `","alt_text":"Crew on deck"}`, nil},
		{"gallery alt required", &GalleryImage{}, `{"image_file_id":"` + id + `","alt_text":"  "}`, []string{"alt_text"}},
		{"testimonial ok", &Testimonial{}, `{"name":"Ari","quote":"Great","rating":5}`, nil},
		{"testimonial rating", &Testimonial{}, `{"name":"Ari","quote":"Great","rating":6}`, []string{"rating"}},
		{"testimonial rating missing", &Testimonial{}, `{"name":"Ari","quote":"Great"}`, []string{"rating"}},
		{"testimonial quote", &Testimonial{}, `{"name":"Ari","quote":"","rating":3}`, []string{"quote"}},
		{"logo ok", &ClientLogo{}, `{"name":"Acme","logo_file_id":"` + id + `","url":"https://acme.example.com"}`, nil},
		{"logo url must be https", &ClientLogo{}, `{"name":"Acme","logo_file_id":"` + id + `","url":"http://acme.example.com"}`, []string{"url"}},
		{"logo file required", &ClientLogo{}, `{"name":"Acme"}`, []string{"logo_file_id"}},
		{"stats ok", &StatsItem{}, `{"number":0,"suffix":"+","label":"Crew"}`, nil},
		{"stats number required", &StatsItem{}, `{"label":"Crew"}`, []string{"number"}},
		{"stats negative", &StatsItem{}, `{"number":-5,"label":"Crew"}`, []string{"number"}},
		{"language ok", &Language{}, `{"name":"Chinese","code":"zh-CN","flag_code":"CN"}`, nil},
		{"language code", &Language{}, `{"name":"Chinese","code":"Chinese","flag_code":"cn"}`, []string{"code"}},
		{"language needs flag", &Language{}, `{"name":"Japanese","code":"ja"}`, []string{"flag_code"}},
		{"language flag code", &Language{}, `{"name":"Japanese","code":"ja","flag_code":"xx"}`, []string{"flag_code"}},
		{"language uploaded flag", &Language{}, `{"name":"Japanese","code":"ja","flag_file_id":"` + id + `"}`, nil},
		{"news ok", &News{}, `{"title":"Crew change","status":"published","published_at":"2026-10-21T08:00:00Z"}`, nil},
		{"news excerpt", &News{}, `{"title":"Crew change","excerpt":"` + strings.Repeat("x", 301) + `"}`, []string{"excerpt"}},
		{"news status", &News{}, `{"title":"Crew change","status":"scheduled"}`, []string{"status"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fields := decodeInto(t, c.dst, c.body)
			if c.keys == nil {
				assert.Empty(t, fields)
				return
			}
			assert.Len(t, fields, len(c.keys), fields)
			for _, k := range c.keys {
				assert.Contains(t, fields, k)
			}
		})
	}
}

func TestLanguageFlagCodeIsLowercased(t *testing.T) {
	var l Language
	require.Empty(t, decodeInto(t, &l, `{"name":"Indonesian","code":"id","flag_code":"ID"}`))
	assert.Equal(t, "id", l.Model().FlagCode)
	assert.True(t, l.Model().IsActive, "languages are active unless said otherwise")
}

func TestGalleryBulkValidatesEveryItem(t *testing.T) {
	id := uuid.NewString()
	var ok GalleryBulk
	require.Empty(t, decodeInto(t, &ok, `{"items":[{"image_file_id":"`+id+`","caption":"A"},{"image_file_id":"`+id+`"}],"show_on_home":true}`))
	models := ok.Model()
	require.Len(t, models, 2)
	assert.True(t, models[1].ShowOnHome)
	assert.Equal(t, "", models[1].AltText, "the service fills a missing alt text")

	var empty GalleryBulk
	assert.Contains(t, decodeInto(t, &empty, `{"items":[]}`), "items")

	var bad GalleryBulk
	fields := decodeInto(t, &bad, `{"items":[{"caption":"A"}]}`)
	assert.Contains(t, fields, "items.0.image_file_id")
}

func TestMenuRequestValidatesLinksAndDepth(t *testing.T) {
	page := uuid.NewString()
	cases := []struct {
		name string
		body string
		keys []string
	}{
		{"ok", `{"items":[{"label":"About","page_id":"` + page + `","children":[{"label":"Safety","url":"/safety"},{"label":"Docs","url":"https://example.com","open_in_new_tab":true}]},{"label":"Mail","url":"mailto:crew@example.com"}]}`, nil},
		{"empty menu", `{"items":[]}`, nil},
		{"items required", `{}`, []string{"items"}},
		{"both page and url", `{"items":[{"label":"X","page_id":"` + page + `","url":"/x"}]}`, []string{"items.0.url"}},
		{"neither", `{"items":[{"label":"X"}]}`, []string{"items.0.page_id"}},
		{"bad url", `{"items":[{"label":"X","url":"javascript:alert(1)"}]}`, []string{"items.0.url"}},
		{"label required", `{"items":[{"label":"","url":"/x"}]}`, []string{"items.0.label"}},
		{"third level", `{"items":[{"label":"A","url":"/a","children":[{"label":"B","url":"/b","children":[{"label":"C"}]}]}]}`, []string{"items.0.children.0.children"}},
		{"child link", `{"items":[{"label":"A","url":"/a","children":[{"label":"B","url":"ftp://b"}]}]}`, []string{"items.0.children.0.url"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var m Menu
			fields := decodeInto(t, &m, c.body)
			if c.keys == nil {
				assert.Empty(t, fields)
				return
			}
			assert.Len(t, fields, len(c.keys), fields)
			for _, k := range c.keys {
				assert.Contains(t, fields, k)
			}
		})
	}
}

func TestMenuRequestModelNestsChildren(t *testing.T) {
	var m Menu
	require.Empty(t, decodeInto(t, &m, `{"items":[{"label":"A","url":"/a","children":[{"label":"B","url":"/b"}]}]}`))
	items := m.Model()
	require.Len(t, items, 1)
	require.Len(t, items[0].Children, 1)
	assert.Equal(t, "/b", items[0].Children[0].URL)
}
