package resource

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

type prefixURLs struct{}

func (prefixURLs) URL(key string) string {
	rest, ok := strings.CutPrefix(key, "public/")
	if !ok {
		return ""
	}
	return "https://api.test/media/" + rest
}

func TestPublicImageListsVariantsThenOriginalForSrcset(t *testing.T) {
	img := &model.Image{StorageKey: "public/news/a.webp", MimeType: "image/webp", Width: 1600, Height: 900, Variants: []model.FileVariant{
		{Width: 480, Height: 270, StorageKey: "public/news/a-480.webp"},
		{Width: 960, Height: 540, StorageKey: "public/news/a-960.webp"},
	}}
	got := NewPublicImage(img, prefixURLs{}, "Crew boarding")
	assert.Equal(t, "https://api.test/media/news/a.webp", got.URL)
	assert.Equal(t, "Crew boarding", got.Alt)
	assert.Equal(t, []ImageSource{
		{URL: "https://api.test/media/news/a-480.webp", Width: 480, Height: 270},
		{URL: "https://api.test/media/news/a-960.webp", Width: 960, Height: 540},
		{URL: "https://api.test/media/news/a.webp", Width: 1600, Height: 900},
	}, got.Sources)
}

func TestPublicImageIsNilForMissingOrPrivateFiles(t *testing.T) {
	assert.Nil(t, NewPublicImage(nil, prefixURLs{}, ""))
	assert.Nil(t, NewPublicImage(&model.Image{StorageKey: "crews/x.webp"}, prefixURLs{}, ""))
}

func TestPublicImageWithoutDimensionsHasNoSources(t *testing.T) {
	got := NewPublicImage(&model.Image{StorageKey: "public/logo/a.svg", MimeType: "image/svg+xml"}, prefixURLs{}, "")
	assert.Empty(t, got.Sources)
}
