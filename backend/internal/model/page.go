package model

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Page system keys.
const (
	SystemHome    = "home"
	SystemNews    = "news"
	SystemCareer  = "career"
	SystemContact = "contact"
	SystemGallery = "gallery"
)

// SystemPath returns the fixed public path of a system page key, or "" for an unknown key.
func SystemPath(key string) string {
	switch key {
	case SystemHome:
		return "/"
	case SystemNews, SystemCareer, SystemContact, SystemGallery:
		return "/" + key
	}
	return ""
}

// Page is a CMS page with its builder blocks.
type Page struct {
	ID              uuid.UUID
	Title           string
	Slug            string
	SystemKey       *string
	IntroHTML       string
	Blocks          []Block
	SEOTitle        string
	MetaDescription string
	OGImageFileID   *uuid.UUID
	Status          string
	InHeaderMenu    bool
	InFooterMenu    bool
	MenuOrder       int
	IsSystem        bool
	PublishedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	OGImage         *Image
}

// Path is the public path of the page.
func (p Page) Path() string {
	if p.SystemKey != nil {
		return SystemPath(*p.SystemKey)
	}
	return "/" + p.Slug
}

// Block types of the page builder.
const (
	BlockRichText  = "rich_text"
	BlockImageText = "image_text"
	BlockGallery   = "gallery"
	BlockVideo     = "video"
	BlockCTA       = "cta"
	BlockStats     = "stats"
	BlockFAQ       = "faq"
	BlockCardGrid  = "card_grid"
)

// Block is one builder block; Settings holds the typed settings of Type.
type Block struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Hidden   bool   `json:"hidden"`
	Settings any    `json:"settings"`
}

// RichTextSettings are the settings of a rich text block.
type RichTextSettings struct {
	HTML string `json:"html"`
}

// ImageTextSettings are the settings of an image and text split block.
type ImageTextSettings struct {
	ImageFileID   *uuid.UUID `json:"image_file_id"`
	Heading       string     `json:"heading"`
	TextHTML      string     `json:"text_html"`
	ButtonLabel   string     `json:"button_label"`
	ButtonLink    string     `json:"button_link"`
	ImagePosition string     `json:"image_position"`
}

// GallerySettings are the settings of a gallery block; ImageIDs are gallery image record IDs.
type GallerySettings struct {
	ImageIDs []uuid.UUID `json:"image_ids"`
	Columns  int         `json:"columns"`
	Lightbox bool        `json:"lightbox"`
}

// VideoSettings are the settings of a video block; Provider and VideoID are derived from URL on save.
type VideoSettings struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Provider string `json:"provider"`
	VideoID  string `json:"video_id"`
}

// CTASettings are the settings of a call to action banner.
type CTASettings struct {
	Heading               string     `json:"heading"`
	Text                  string     `json:"text"`
	ButtonLabel           string     `json:"button_label"`
	ButtonLink            string     `json:"button_link"`
	BackgroundImageFileID *uuid.UUID `json:"background_image_file_id"`
	BackgroundColor       string     `json:"background_color"`
}

// StatsItemSetting is one counter of a stats block.
type StatsItemSetting struct {
	Number int    `json:"number"`
	Suffix string `json:"suffix"`
	Label  string `json:"label"`
}

// StatsSettings are the settings of a stats block.
type StatsSettings struct {
	Items []StatsItemSetting `json:"items"`
}

// FAQItem is one question of a FAQ block.
type FAQItem struct {
	Question   string `json:"question"`
	AnswerHTML string `json:"answer_html"`
}

// FAQSettings are the settings of a FAQ block.
type FAQSettings struct {
	Items []FAQItem `json:"items"`
}

// Card is one card of a card grid.
type Card struct {
	Icon        string     `json:"icon"`
	ImageFileID *uuid.UUID `json:"image_file_id"`
	Title       string     `json:"title"`
	Text        string     `json:"text"`
	Link        string     `json:"link"`
}

// CardGridSettings are the settings of a card grid block.
type CardGridSettings struct {
	Columns int    `json:"columns"`
	Cards   []Card `json:"cards"`
}

// NewBlockSettings returns an empty settings value for blockType.
func NewBlockSettings(blockType string) (any, bool) {
	switch blockType {
	case BlockRichText:
		return &RichTextSettings{}, true
	case BlockImageText:
		return &ImageTextSettings{}, true
	case BlockGallery:
		return &GallerySettings{}, true
	case BlockVideo:
		return &VideoSettings{}, true
	case BlockCTA:
		return &CTASettings{}, true
	case BlockStats:
		return &StatsSettings{}, true
	case BlockFAQ:
		return &FAQSettings{}, true
	case BlockCardGrid:
		return &CardGridSettings{}, true
	}
	return nil, false
}

// UnmarshalJSON decodes a stored block into its typed settings.
func (b *Block) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID       string          `json:"id"`
		Type     string          `json:"type"`
		Hidden   bool            `json:"hidden"`
		Settings json.RawMessage `json:"settings"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("decode block: %w", err)
	}
	settings, ok := NewBlockSettings(raw.Type)
	if !ok {
		return fmt.Errorf("decode block: unknown type %q", raw.Type)
	}
	if len(raw.Settings) > 0 {
		if err := json.Unmarshal(raw.Settings, settings); err != nil {
			return fmt.Errorf("decode %s settings: %w", raw.Type, err)
		}
	}
	*b = Block{ID: raw.ID, Type: raw.Type, Hidden: raw.Hidden, Settings: settings}
	return nil
}
