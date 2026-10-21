package request

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/video"
)

var (
	blockIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	iconPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)
)

const (
	maxBlockHTML    = 100_000
	maxGalleryItems = 60
	maxStatsItems   = 12
	maxFAQItems     = 50
	maxCards        = 24
	defaultColumns  = 3
)

type blockEnvelope struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Hidden   bool            `json:"hidden"`
	Settings json.RawMessage `json:"settings"`
}

// ParseBlocks decodes and validates the builder blocks, reporting problems under keys such as blocks.3.settings.columns.
func ParseBlocks(raws []json.RawMessage, f FieldErrors) []model.Block {
	if len(raws) > MaxBlocks {
		f.set("blocks", fmt.Sprintf("A page may have at most %d blocks.", MaxBlocks))
		return nil
	}
	blocks := make([]model.Block, 0, len(raws))
	seen := make(map[string]bool, len(raws))
	for i, raw := range raws {
		b, ok := parseBlock(i, raw, seen, f)
		if ok {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

func parseBlock(i int, raw json.RawMessage, seen map[string]bool, f FieldErrors) (model.Block, bool) {
	prefix := fmt.Sprintf("blocks.%d.", i)
	var env blockEnvelope
	if !strictDecode(raw, &env, prefix, f) {
		return model.Block{}, false
	}
	if !blockIDPattern.MatchString(env.ID) {
		f.set(prefix+"id", "The block ID must be 1 to 64 letters, digits, - or _.")
	} else if seen[env.ID] {
		f.set(prefix+"id", "The block ID must be unique within the page.")
	}
	seen[env.ID] = true
	settings, known := model.NewBlockSettings(env.Type)
	if !known {
		f.set(prefix+"type", "The block type is not supported.")
		return model.Block{}, false
	}
	if len(bytes.TrimSpace(env.Settings)) == 0 || bytes.Equal(bytes.TrimSpace(env.Settings), []byte("null")) {
		env.Settings = json.RawMessage(`{}`)
	}
	sp := prefix + "settings."
	if !strictDecode(env.Settings, settings, sp, f) {
		return model.Block{}, false
	}
	e := blockErrors{f: f, prefix: sp}
	switch v := settings.(type) {
	case *model.RichTextSettings:
		e.richText(v)
	case *model.ImageTextSettings:
		e.imageText(v)
	case *model.GallerySettings:
		e.gallery(v)
	case *model.VideoSettings:
		e.video(v)
	case *model.CTASettings:
		e.cta(v)
	case *model.StatsSettings:
		e.stats(v)
	case *model.FAQSettings:
		e.faq(v)
	case *model.CardGridSettings:
		e.cardGrid(v)
	}
	return model.Block{ID: env.ID, Type: env.Type, Hidden: env.Hidden, Settings: settings}, true
}

func strictDecode(raw []byte, dst any, prefix string, f FieldErrors) bool {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	if err == nil {
		return true
	}
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &typeErr):
		f.set(prefix+typeErr.Field, "The value has the wrong type.")
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		name := strings.Trim(strings.TrimPrefix(err.Error(), "json: unknown field "), `"`)
		f.set(prefix+name, "This field is not supported.")
	default:
		f.set(strings.TrimSuffix(prefix, "."), "The value is not valid JSON of the expected shape.")
	}
	return false
}

type blockErrors struct {
	f      FieldErrors
	prefix string
}

func (e blockErrors) key(name string) string { return e.prefix + name }

func (e blockErrors) required(name, value string, max int) {
	value = strings.TrimSpace(value)
	if value == "" {
		e.f.set(e.key(name), "This field is required.")
		return
	}
	e.maxLen(name, value, max)
}

func (e blockErrors) maxLen(name, value string, max int) {
	if utf8.RuneCountInString(value) > max {
		e.f.set(e.key(name), fmt.Sprintf("This field may not exceed %d characters.", max))
	}
}

func (e blockErrors) columns(value *int) {
	if *value == 0 {
		*value = defaultColumns
	}
	if *value < 2 || *value > 4 {
		e.f.set(e.key("columns"), "The columns must be between 2 and 4.")
	}
}

func (e blockErrors) button(label, href *string) {
	*label, *href = strings.TrimSpace(*label), strings.TrimSpace(*href)
	e.maxLen("button_label", *label, 60)
	e.link("button_link", *href)
	if *label != "" && *href == "" {
		e.f.set(e.key("button_link"), "The button link is required when the button has a label.")
	}
	if *href != "" && *label == "" {
		e.f.set(e.key("button_label"), "The button label is required when the button has a link.")
	}
}

func (e blockErrors) link(name, value string) {
	sub := FieldErrors{}
	sub.Link(name, value)
	if msg, bad := sub[name]; bad {
		e.f.set(e.key(name), msg)
	}
}

func (e blockErrors) richText(v *model.RichTextSettings) {
	if strings.TrimSpace(v.HTML) == "" {
		e.f.set(e.key("html"), "This field is required.")
		return
	}
	e.maxLen("html", v.HTML, maxBlockHTML)
}

func (e blockErrors) imageText(v *model.ImageTextSettings) {
	if v.ImageFileID == nil {
		e.f.set(e.key("image_file_id"), "The image is required.")
	}
	v.Heading = strings.TrimSpace(v.Heading)
	e.maxLen("heading", v.Heading, 160)
	e.maxLen("text_html", v.TextHTML, maxBlockHTML)
	e.button(&v.ButtonLabel, &v.ButtonLink)
	if v.ImagePosition == "" {
		v.ImagePosition = "left"
	}
	if v.ImagePosition != "left" && v.ImagePosition != "right" {
		e.f.set(e.key("image_position"), "The image position must be one of: left, right.")
	}
}

func (e blockErrors) gallery(v *model.GallerySettings) {
	e.columns(&v.Columns)
	switch {
	case len(v.ImageIDs) == 0:
		e.f.set(e.key("image_ids"), "Select at least one gallery image.")
	case len(v.ImageIDs) > maxGalleryItems:
		e.f.set(e.key("image_ids"), fmt.Sprintf("Select at most %d gallery images.", maxGalleryItems))
	}
	seen := map[uuid.UUID]bool{}
	for i, id := range v.ImageIDs {
		if id == uuid.Nil || seen[id] {
			e.f.set(e.key(fmt.Sprintf("image_ids.%d", i)), "Each gallery image must be selected once.")
		}
		seen[id] = true
	}
}

func (e blockErrors) video(v *model.VideoSettings) {
	v.URL = strings.TrimSpace(v.URL)
	v.Title = strings.TrimSpace(v.Title)
	e.maxLen("title", v.Title, 160)
	ref, ok := video.Parse(v.URL)
	if !ok {
		e.f.set(e.key("url"), "The URL must be a YouTube or Vimeo video link.")
	}
	v.Provider, v.VideoID = ref.Provider, ref.ID
}

func (e blockErrors) cta(v *model.CTASettings) {
	v.Heading = strings.TrimSpace(v.Heading)
	v.Text = strings.TrimSpace(v.Text)
	v.BackgroundColor = strings.TrimSpace(v.BackgroundColor)
	e.required("heading", v.Heading, 160)
	e.maxLen("text", v.Text, 500)
	e.button(&v.ButtonLabel, &v.ButtonLink)
	if v.BackgroundColor != "" && !hexColor.MatchString(v.BackgroundColor) {
		e.f.set(e.key("background_color"), "The background color must be a hex color such as #0B2545.")
	}
}

func (e blockErrors) stats(v *model.StatsSettings) {
	if len(v.Items) == 0 || len(v.Items) > maxStatsItems {
		e.f.set(e.key("items"), fmt.Sprintf("Add between 1 and %d items.", maxStatsItems))
		return
	}
	for i := range v.Items {
		it := &v.Items[i]
		it.Suffix, it.Label = strings.TrimSpace(it.Suffix), strings.TrimSpace(it.Label)
		item := blockErrors{f: e.f, prefix: e.key(fmt.Sprintf("items.%d.", i))}
		if it.Number < 0 || it.Number > MaxStatsNumber {
			e.f.set(item.key("number"), fmt.Sprintf("The number must be between 0 and %d.", MaxStatsNumber))
		}
		item.maxLen("suffix", it.Suffix, 10)
		item.required("label", it.Label, 80)
	}
}

func (e blockErrors) faq(v *model.FAQSettings) {
	if len(v.Items) == 0 || len(v.Items) > maxFAQItems {
		e.f.set(e.key("items"), fmt.Sprintf("Add between 1 and %d items.", maxFAQItems))
		return
	}
	for i := range v.Items {
		it := &v.Items[i]
		it.Question = strings.TrimSpace(it.Question)
		item := blockErrors{f: e.f, prefix: e.key(fmt.Sprintf("items.%d.", i))}
		item.required("question", it.Question, 300)
		if strings.TrimSpace(it.AnswerHTML) == "" {
			e.f.set(item.key("answer_html"), "This field is required.")
		}
		item.maxLen("answer_html", it.AnswerHTML, 20_000)
	}
}

func (e blockErrors) cardGrid(v *model.CardGridSettings) {
	e.columns(&v.Columns)
	if len(v.Cards) == 0 || len(v.Cards) > maxCards {
		e.f.set(e.key("cards"), fmt.Sprintf("Add between 1 and %d cards.", maxCards))
		return
	}
	for i := range v.Cards {
		c := &v.Cards[i]
		c.Icon, c.Title, c.Text, c.Link = strings.TrimSpace(c.Icon), strings.TrimSpace(c.Title), strings.TrimSpace(c.Text), strings.TrimSpace(c.Link)
		card := blockErrors{f: e.f, prefix: e.key(fmt.Sprintf("cards.%d.", i))}
		card.required("title", c.Title, 120)
		card.maxLen("text", c.Text, 500)
		card.link("link", c.Link)
		if c.Icon != "" && !iconPattern.MatchString(c.Icon) {
			e.f.set(card.key("icon"), "The icon must be a lowercase icon name such as anchor or ship-wheel.")
		}
	}
}
