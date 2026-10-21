package request

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

func pageBody(t *testing.T, blocks string) *Page {
	t.Helper()
	var p Page
	body := fmt.Sprintf(`{"title":"About Us","blocks":%s}`, blocks)
	require.NoError(t, json.Unmarshal([]byte(body), &p))
	return &p
}

func TestPageBlocksAreValidatedPerType(t *testing.T) {
	img := uuid.NewString()
	gal := uuid.NewString()
	cases := []struct {
		name    string
		block   string
		invalid map[string]string
	}{
		{"rich text", `{"id":"a","type":"rich_text","settings":{"html":"<p>Hi</p>"}}`, nil},
		{"rich text needs html", `{"id":"a","type":"rich_text","settings":{"html":" "}}`, map[string]string{"blocks.0.settings.html": ""}},
		{"image text", `{"id":"a","type":"image_text","settings":{"image_file_id":"` + img + `","heading":"H","button_label":"Go","button_link":"/career","image_position":"right"}}`, nil},
		{"image text needs image", `{"id":"a","type":"image_text","settings":{"heading":"H"}}`, map[string]string{"blocks.0.settings.image_file_id": ""}},
		{"image text position", `{"id":"a","type":"image_text","settings":{"image_file_id":"` + img + `","image_position":"top"}}`, map[string]string{"blocks.0.settings.image_position": ""}},
		{"button needs link", `{"id":"a","type":"image_text","settings":{"image_file_id":"` + img + `","button_label":"Go"}}`, map[string]string{"blocks.0.settings.button_link": ""}},
		{"button link scheme", `{"id":"a","type":"cta","settings":{"heading":"H","button_label":"Go","button_link":"javascript:alert(1)"}}`, map[string]string{"blocks.0.settings.button_link": ""}},
		{"gallery", `{"id":"a","type":"gallery","settings":{"image_ids":["` + gal + `"],"columns":4,"lightbox":true}}`, nil},
		{"gallery columns", `{"id":"a","type":"gallery","settings":{"image_ids":["` + gal + `"],"columns":5}}`, map[string]string{"blocks.0.settings.columns": ""}},
		{"gallery empty", `{"id":"a","type":"gallery","settings":{"image_ids":[],"columns":3}}`, map[string]string{"blocks.0.settings.image_ids": ""}},
		{"gallery repeated image", `{"id":"a","type":"gallery","settings":{"image_ids":["` + gal + `","` + gal + `"]}}`, map[string]string{"blocks.0.settings.image_ids.1": ""}},
		{"video", `{"id":"a","type":"video","settings":{"url":"https://youtu.be/dQw4w9WgXcQ","title":"Intro"}}`, nil},
		{"video other host", `{"id":"a","type":"video","settings":{"url":"https://example.com/v/1"}}`, map[string]string{"blocks.0.settings.url": ""}},
		{"cta", `{"id":"a","type":"cta","settings":{"heading":"Join","background_color":"#0B2545"}}`, nil},
		{"cta color", `{"id":"a","type":"cta","settings":{"heading":"Join","background_color":"navy"}}`, map[string]string{"blocks.0.settings.background_color": ""}},
		{"cta heading", `{"id":"a","type":"cta","settings":{"heading":""}}`, map[string]string{"blocks.0.settings.heading": ""}},
		{"stats", `{"id":"a","type":"stats","settings":{"items":[{"number":1200,"suffix":"+","label":"Crew"}]}}`, nil},
		{"stats label", `{"id":"a","type":"stats","settings":{"items":[{"number":-1,"label":""}]}}`, map[string]string{"blocks.0.settings.items.0.number": "", "blocks.0.settings.items.0.label": ""}},
		{"faq", `{"id":"a","type":"faq","settings":{"items":[{"question":"Fees?","answer_html":"<p>None</p>"}]}}`, nil},
		{"faq answer", `{"id":"a","type":"faq","settings":{"items":[{"question":"Fees?","answer_html":""}]}}`, map[string]string{"blocks.0.settings.items.0.answer_html": ""}},
		{"card grid", `{"id":"a","type":"card_grid","settings":{"columns":3,"cards":[{"icon":"ship-wheel","title":"Crewing","link":"/services"}]}}`, nil},
		{"card grid card", `{"id":"a","type":"card_grid","settings":{"columns":3,"cards":[{"icon":"Bad Icon","title":"","link":"http://x.com"}]}}`, map[string]string{
			"blocks.0.settings.cards.0.icon": "", "blocks.0.settings.cards.0.title": "", "blocks.0.settings.cards.0.link": "",
		}},
		{"unknown type", `{"id":"a","type":"carousel","settings":{}}`, map[string]string{"blocks.0.type": ""}},
		{"unknown settings field", `{"id":"a","type":"rich_text","settings":{"html":"<p>x</p>","color":"red"}}`, map[string]string{"blocks.0.settings.color": ""}},
		{"unknown block field", `{"id":"a","type":"rich_text","settings":{"html":"<p>x</p>"},"extra":1}`, map[string]string{"blocks.0.extra": ""}},
		{"wrong type", `{"id":"a","type":"gallery","settings":{"image_ids":"nope"}}`, map[string]string{"blocks.0.settings.image_ids": ""}},
		{"bad id", `{"id":"has space","type":"rich_text","settings":{"html":"<p>x</p>"}}`, map[string]string{"blocks.0.id": ""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fields := pageBody(t, "["+c.block+"]").Validate()
			if c.invalid == nil {
				assert.Empty(t, fields)
				return
			}
			for key := range c.invalid {
				assert.Contains(t, fields, key)
			}
			assert.Len(t, fields, len(c.invalid))
		})
	}
}

func TestPageBlockIDsMustBeUnique(t *testing.T) {
	block := `{"id":"a","type":"rich_text","settings":{"html":"<p>x</p>"}}`
	fields := pageBody(t, "["+block+","+block+"]").Validate()
	assert.Contains(t, fields, "blocks.1.id")
}

func TestPageAllowsAtMostOneHundredBlocks(t *testing.T) {
	blocks := make([]string, 0, 101)
	for i := range 101 {
		blocks = append(blocks, fmt.Sprintf(`{"id":"b%d","type":"rich_text","settings":{"html":"<p>x</p>"}}`, i))
	}
	assert.Contains(t, pageBody(t, "["+strings.Join(blocks, ",")+"]").Validate(), "blocks")
	assert.Empty(t, pageBody(t, "["+strings.Join(blocks[:100], ",")+"]").Validate())
}

func TestPageVideoBlockStoresProviderAndVideoID(t *testing.T) {
	p := pageBody(t, `[{"id":"v","type":"video","hidden":true,"settings":{"url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","title":"Intro","provider":"vimeo","video_id":"1"}}]`)
	require.Empty(t, p.Validate())
	blocks := p.Model().Blocks
	require.Len(t, blocks, 1)
	assert.True(t, blocks[0].Hidden)
	v, ok := blocks[0].Settings.(*model.VideoSettings)
	require.True(t, ok)
	assert.Equal(t, "youtube", v.Provider, "client supplied provider is ignored")
	assert.Equal(t, "dQw4w9WgXcQ", v.VideoID)
}

func TestPageBlocksDefaultColumnsAndPosition(t *testing.T) {
	gal := uuid.NewString()
	p := pageBody(t, `[{"id":"g","type":"gallery","settings":{"image_ids":["`+gal+`"]}},{"id":"i","type":"image_text","settings":{"image_file_id":"`+uuid.NewString()+`"}}]`)
	require.Empty(t, p.Validate())
	blocks := p.Model().Blocks
	assert.Equal(t, 3, blocks[0].Settings.(*model.GallerySettings).Columns)
	assert.Equal(t, "left", blocks[1].Settings.(*model.ImageTextSettings).ImagePosition)
}

func TestPageFieldsAreValidated(t *testing.T) {
	var p Page
	require.NoError(t, json.Unmarshal([]byte(`{"title":"","status":"archived","menu_order":-1,"meta_description":"`+strings.Repeat("x", 321)+`"}`), &p))
	fields := p.Validate()
	assert.Contains(t, fields, "title")
	assert.Contains(t, fields, "status")
	assert.Contains(t, fields, "menu_order")
	assert.Contains(t, fields, "meta_description")

	var ok Page
	require.NoError(t, json.Unmarshal([]byte(`{"title":" About "}`), &ok))
	require.Empty(t, ok.Validate())
	assert.Equal(t, "About", ok.Model().Title)
	assert.Equal(t, model.StatusDraft, ok.Model().Status, "status defaults to draft")
}
