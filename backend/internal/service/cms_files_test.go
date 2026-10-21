package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicFileCheckerAppliesTheMimeRulesOfEachKind(t *testing.T) {
	files := newFakeFiles()
	webp := files.add("image/webp", "photo.jpg")
	svg := files.add("image/svg+xml", "mark.svg")
	pdf := files.add("application/pdf", "cv.pdf")
	missing := uuid.New()
	checker := NewPublicFileChecker(files)

	cases := []struct {
		name    string
		id      uuid.UUID
		kind    PublicFileKind
		invalid bool
	}{
		{"image accepts webp", webp, ImageFile, false},
		{"image rejects svg", svg, ImageFile, true},
		{"logo accepts webp", webp, LogoFile, false},
		{"logo accepts svg", svg, LogoFile, false},
		{"logo rejects pdf", pdf, LogoFile, true},
		{"flag accepts svg", svg, FlagFile, false},
		{"flag rejects webp", webp, FlagFile, true},
		{"unknown file", missing, ImageFile, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := checker.Check(context.Background(), FileRef{Field: "file", ID: &c.id, Kind: c.kind})
			if c.invalid {
				assert.Contains(t, validationFields(t, err), "file")
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestPublicFileCheckerSkipsEmptyReferencesAndReportsEveryBadField(t *testing.T) {
	files := newFakeFiles()
	ok := files.add("image/webp", "a.jpg")
	bad := uuid.New()
	checker := NewPublicFileChecker(files)

	infos, err := checker.Check(context.Background(), FileRef{Field: "a", Kind: ImageFile}, FileRef{Field: "b", ID: &ok, Kind: ImageFile})
	require.NoError(t, err)
	assert.Equal(t, "a.jpg", infos[ok].OriginalName)

	_, err = checker.Check(context.Background(), FileRef{Field: "x", ID: &bad, Kind: ImageFile}, FileRef{Field: "y", ID: &bad, Kind: LogoFile})
	assert.Len(t, validationFields(t, err), 2)
}
