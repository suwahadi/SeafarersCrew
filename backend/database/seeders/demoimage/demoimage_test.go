package demoimage

import (
	"bytes"
	"crypto/sha256"
	"image"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decode(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	return img
}

func TestSceneIsALandscapeImageOfTheDocumentedSize(t *testing.T) {
	for index := range SceneCount() {
		data, err := Scene(index)
		require.NoError(t, err)
		b := decode(t, data).Bounds()
		assert.Equal(t, SceneWidth, b.Dx(), "scene %d", index)
		assert.Equal(t, SceneHeight, b.Dy(), "scene %d", index)
	}
}

func TestLogoIsSquare(t *testing.T) {
	data, err := Logo(3, "Samudera Nusantara")
	require.NoError(t, err)
	b := decode(t, data).Bounds()
	assert.Equal(t, LogoSize, b.Dx())
	assert.Equal(t, LogoSize, b.Dy())
}

func TestImagesAreDeterministic(t *testing.T) {
	tests := []struct {
		name string
		draw func() ([]byte, error)
	}{
		{"scene", func() ([]byte, error) { return Scene(7) }},
		{"logo", func() ([]byte, error) { return Logo(2, "Bahari") }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			first, err := tc.draw()
			require.NoError(t, err)
			second, err := tc.draw()
			require.NoError(t, err)
			assert.Equal(t, first, second)
		})
	}
}

func TestScenesAreDistinct(t *testing.T) {
	seen := map[[32]byte]int{}
	for index := range 24 {
		data, err := Scene(index)
		require.NoError(t, err)
		sum := sha256.Sum256(data)
		prev, dup := seen[sum]
		assert.False(t, dup, "scene %d equals scene %d", index, prev)
		seen[sum] = index
	}
}

func TestLogosAreDistinct(t *testing.T) {
	names := []string{"Samudera Nusantara", "Bahari Lintas", "Meridian Marine", "Karya Tirta", "Fjord Shipping", "Orion Offshore", "Pelita Laut", "Garuda Tankers"}
	seen := map[[32]byte]string{}
	for i, name := range names {
		data, err := Logo(i, name)
		require.NoError(t, err)
		sum := sha256.Sum256(data)
		prev, dup := seen[sum]
		assert.False(t, dup, "logo %q equals %q", name, prev)
		seen[sum] = name
	}
}

func TestLogoWithoutLettersStillRenders(t *testing.T) {
	data, err := Logo(0, "1234")
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestNegativeIndexIsRejected(t *testing.T) {
	_, err := Scene(-1)
	assert.Error(t, err)
	_, err = Logo(-1, "A")
	assert.Error(t, err)
}

func TestInitialsKeepTheFirstTwoLetters(t *testing.T) {
	tests := []struct{ name, want string }{
		{"Samudera Nusantara", "SN"},
		{"Bahari", "BA"},
		{"pt bahari", "PB"},
		{"7 Seas", "SE"},
		{"", ""},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, initials(tc.name), tc.name)
	}
}
