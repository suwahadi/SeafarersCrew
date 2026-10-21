package request

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/flags"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/link"
)

var languageCode = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z]{2,4})?$`)

// MaxStatsNumber is the largest counter value accepted.
const MaxStatsNumber = 1_000_000_000

// Link checks an optional relative path, https URL, mailto address or tel number.
func (f FieldErrors) Link(field, value string) {
	if value != "" && !link.Valid(value) {
		f.set(field, "The "+label(field)+" must be a relative path starting with /, an https URL, a mailto: or a tel: link.")
	}
}

// ButtonPair checks that a button has both a label and a link, or neither.
func (f FieldErrors) ButtonPair(labelField, linkField, buttonLabel, buttonLink string) {
	if buttonLabel != "" && buttonLink == "" {
		f.set(linkField, "The button link is required when the button has a label.")
	}
	if buttonLink != "" && buttonLabel == "" {
		f.set(labelField, "The button label is required when the button has a link.")
	}
}

// RequiredID checks that a mandatory identifier was sent.
func (f FieldErrors) RequiredID(field string, id uuid.UUID) {
	if id == uuid.Nil {
		f.set(field, "The "+label(field)+" is required.")
	}
}

// HeroSlide is the body of POST and PUT /hero-slides.
type HeroSlide struct {
	ImageFileID       uuid.UUID  `json:"image_file_id"`
	MobileImageFileID *uuid.UUID `json:"mobile_image_file_id"`
	Heading           string     `json:"heading"`
	Subheading        string     `json:"subheading"`
	ButtonLabel       string     `json:"button_label"`
	ButtonLink        string     `json:"button_link"`
	IsActive          *bool      `json:"is_active"`
}

// Validate implements Validator.
func (r *HeroSlide) Validate() map[string]string {
	trimAll(&r.Heading, &r.Subheading, &r.ButtonLabel, &r.ButtonLink)
	f := FieldErrors{}
	f.RequiredID("image_file_id", r.ImageFileID)
	f.MaxLen("heading", r.Heading, 160)
	f.MaxLen("subheading", r.Subheading, 300)
	f.MaxLen("button_label", r.ButtonLabel, 60)
	f.Link("button_link", r.ButtonLink)
	f.ButtonPair("button_label", "button_link", r.ButtonLabel, r.ButtonLink)
	return f
}

// Model converts the request to a model.
func (r *HeroSlide) Model() model.HeroSlide {
	return model.HeroSlide{
		MasterBase:  model.MasterBase{IsActive: boolOr(r.IsActive, true)},
		ImageFileID: r.ImageFileID, MobileImageFileID: r.MobileImageFileID,
		Heading: r.Heading, Subheading: r.Subheading, ButtonLabel: r.ButtonLabel, ButtonLink: r.ButtonLink,
	}
}

// GalleryImage is the body of POST and PUT /gallery.
type GalleryImage struct {
	ImageFileID uuid.UUID `json:"image_file_id"`
	Caption     string    `json:"caption"`
	AltText     string    `json:"alt_text"`
	ShowOnHome  bool      `json:"show_on_home"`
	IsActive    *bool     `json:"is_active"`
}

// Validate implements Validator.
func (r *GalleryImage) Validate() map[string]string {
	trimAll(&r.Caption, &r.AltText)
	f := FieldErrors{}
	f.RequiredID("image_file_id", r.ImageFileID)
	f.MaxLen("caption", r.Caption, 300)
	f.Required("alt_text", r.AltText, 200)
	return f
}

// Model converts the request to a model.
func (r *GalleryImage) Model() model.GalleryImage {
	return model.GalleryImage{
		MasterBase:  model.MasterBase{IsActive: boolOr(r.IsActive, true)},
		ImageFileID: r.ImageFileID, Caption: r.Caption, AltText: r.AltText, ShowOnHome: r.ShowOnHome,
	}
}

// GalleryBulkItem is one image of a bulk gallery upload.
type GalleryBulkItem struct {
	ImageFileID uuid.UUID `json:"image_file_id"`
	Caption     string    `json:"caption"`
	AltText     string    `json:"alt_text"`
}

// GalleryBulk is the body of POST /gallery/bulk.
type GalleryBulk struct {
	Items      []GalleryBulkItem `json:"items"`
	ShowOnHome bool              `json:"show_on_home"`
	IsActive   *bool             `json:"is_active"`
}

// Validate implements Validator.
func (r *GalleryBulk) Validate() map[string]string {
	f := FieldErrors{}
	if len(r.Items) == 0 {
		f.set("items", "At least one image is required.")
		return f
	}
	if len(r.Items) > 50 {
		f.set("items", "At most 50 images can be added at once.")
		return f
	}
	for i := range r.Items {
		item := &r.Items[i]
		trimAll(&item.Caption, &item.AltText)
		f.RequiredID(fmt.Sprintf("items.%d.image_file_id", i), item.ImageFileID)
		f.MaxLen(fmt.Sprintf("items.%d.caption", i), item.Caption, 300)
		f.MaxLen(fmt.Sprintf("items.%d.alt_text", i), item.AltText, 200)
	}
	return f
}

// Model converts the request to models.
func (r *GalleryBulk) Model() []model.GalleryImage {
	out := make([]model.GalleryImage, len(r.Items))
	for i, item := range r.Items {
		out[i] = model.GalleryImage{
			MasterBase:  model.MasterBase{IsActive: boolOr(r.IsActive, true)},
			ImageFileID: item.ImageFileID, Caption: item.Caption, AltText: item.AltText, ShowOnHome: r.ShowOnHome,
		}
	}
	return out
}

// Testimonial is the body of POST and PUT /testimonials.
type Testimonial struct {
	Name        string     `json:"name"`
	Position    string     `json:"position"`
	PhotoFileID *uuid.UUID `json:"photo_file_id"`
	Quote       string     `json:"quote"`
	Rating      *int       `json:"rating"`
	IsActive    *bool      `json:"is_active"`
}

// Validate implements Validator.
func (r *Testimonial) Validate() map[string]string {
	trimAll(&r.Name, &r.Position, &r.Quote)
	f := FieldErrors{}
	f.Required("name", r.Name, 120)
	f.MaxLen("position", r.Position, 160)
	f.Required("quote", r.Quote, 1000)
	if r.Rating == nil {
		f.set("rating", "The rating is required.")
	}
	f.Range("rating", r.Rating, 1, 5)
	return f
}

// Model converts the request to a model.
func (r *Testimonial) Model() model.Testimonial {
	rating := 0
	if r.Rating != nil {
		rating = *r.Rating
	}
	return model.Testimonial{
		MasterBase: model.MasterBase{IsActive: boolOr(r.IsActive, true)},
		Name:       r.Name, Position: r.Position, PhotoFileID: r.PhotoFileID, Quote: r.Quote, Rating: rating,
	}
}

// ClientLogo is the body of POST and PUT /client-logos.
type ClientLogo struct {
	Name       string    `json:"name"`
	LogoFileID uuid.UUID `json:"logo_file_id"`
	URL        string    `json:"url"`
	IsActive   *bool     `json:"is_active"`
}

// Validate implements Validator.
func (r *ClientLogo) Validate() map[string]string {
	trimAll(&r.Name, &r.URL)
	f := FieldErrors{}
	f.Required("name", r.Name, 120)
	f.RequiredID("logo_file_id", r.LogoFileID)
	if r.URL != "" && !link.HTTPS(r.URL) {
		f.set("url", "The URL must be a valid https URL.")
	}
	return f
}

// Model converts the request to a model.
func (r *ClientLogo) Model() model.ClientLogo {
	return model.ClientLogo{
		MasterBase: model.MasterBase{IsActive: boolOr(r.IsActive, true)},
		Name:       r.Name, LogoFileID: r.LogoFileID, URL: r.URL,
	}
}

// StatsItem is the body of POST and PUT /stats.
type StatsItem struct {
	Number   *int   `json:"number"`
	Suffix   string `json:"suffix"`
	Label    string `json:"label"`
	IsActive *bool  `json:"is_active"`
}

// Validate implements Validator.
func (r *StatsItem) Validate() map[string]string {
	trimAll(&r.Suffix, &r.Label)
	f := FieldErrors{}
	if r.Number == nil {
		f.set("number", "The number is required.")
	}
	f.Range("number", r.Number, 0, MaxStatsNumber)
	f.MaxLen("suffix", r.Suffix, 10)
	f.Required("label", r.Label, 80)
	return f
}

// Model converts the request to a model.
func (r *StatsItem) Model() model.StatsItem {
	number := 0
	if r.Number != nil {
		number = *r.Number
	}
	return model.StatsItem{
		MasterBase: model.MasterBase{IsActive: boolOr(r.IsActive, true)},
		Number:     number, Suffix: r.Suffix, Label: r.Label,
	}
}

// Language is the body of POST and PUT /languages.
type Language struct {
	Name       string     `json:"name"`
	Code       string     `json:"code"`
	FlagFileID *uuid.UUID `json:"flag_file_id"`
	FlagCode   string     `json:"flag_code"`
	IsActive   *bool      `json:"is_active"`
}

// Validate implements Validator.
func (r *Language) Validate() map[string]string {
	trimAll(&r.Name, &r.Code, &r.FlagCode)
	r.FlagCode = strings.ToLower(r.FlagCode)
	f := FieldErrors{}
	f.Required("name", r.Name, 80)
	f.Required("code", r.Code, 10)
	f.Matches("code", r.Code, languageCode, "The code must be a Google Translate language code such as id, ja or zh-CN.")
	if r.FlagFileID == nil && r.FlagCode == "" {
		f.set("flag_code", "Choose a bundled flag or upload an SVG flag.")
	}
	if r.FlagCode != "" && !flags.Known(r.FlagCode) {
		f.set("flag_code", "The flag code must be a lowercase ISO 3166-1 alpha-2 country code.")
	}
	return f
}

// Model converts the request to a model.
func (r *Language) Model() model.Language {
	return model.Language{
		MasterBase: model.MasterBase{IsActive: boolOr(r.IsActive, true)},
		Name:       r.Name, Code: r.Code, FlagFileID: r.FlagFileID, FlagCode: r.FlagCode,
	}
}
