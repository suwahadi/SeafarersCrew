package resource

import "github.com/suwahadi/seafarerscrew/backend/internal/model"

// MediaURLer turns a public storage key into an absolute URL.
type MediaURLer interface {
	URL(storageKey string) string
}

// ImageSource is one responsive width of a public image.
type ImageSource struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// PublicImage is an image as delivered to the public website.
type PublicImage struct {
	URL      string        `json:"url"`
	MimeType string        `json:"mime_type"`
	Width    int           `json:"width"`
	Height   int           `json:"height"`
	Alt      string        `json:"alt"`
	Sources  []ImageSource `json:"sources"`
}

// NewPublicImage maps img to its public URLs, or nil when there is no public image.
func NewPublicImage(img *model.Image, urls MediaURLer, alt string) *PublicImage {
	if img == nil {
		return nil
	}
	url := urls.URL(img.StorageKey)
	if url == "" {
		return nil
	}
	out := &PublicImage{URL: url, MimeType: img.MimeType, Width: img.Width, Height: img.Height, Alt: alt, Sources: make([]ImageSource, 0, len(img.Variants)+1)}
	for _, v := range img.Variants {
		if u := urls.URL(v.StorageKey); u != "" {
			out.Sources = append(out.Sources, ImageSource{URL: u, Width: v.Width, Height: v.Height})
		}
	}
	if img.Width > 0 {
		out.Sources = append(out.Sources, ImageSource{URL: url, Width: img.Width, Height: img.Height})
	}
	return out
}
