package imaging

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if err := Startup(slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		panic(err)
	}
	code := m.Run()
	Shutdown()
	os.Exit(code)
}

func gradient(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		for y := range h {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 90, A: 255})
		}
	}
	return img
}

// exifSegment builds an APP1 segment with Orientation=6 (rotate 90°) and an ImageDescription carrying a fake GPS note.
func exifSegment() []byte {
	desc := []byte("GPS -6.2,106.8\x00")
	var tiff bytes.Buffer
	tiff.WriteString("II*\x00")
	_ = binary.Write(&tiff, binary.LittleEndian, uint32(8))
	_ = binary.Write(&tiff, binary.LittleEndian, uint16(2))
	_ = binary.Write(&tiff, binary.LittleEndian, []uint16{0x010E, 2})
	_ = binary.Write(&tiff, binary.LittleEndian, []uint32{uint32(len(desc)), 8 + 2 + 24 + 4})
	_ = binary.Write(&tiff, binary.LittleEndian, []uint16{0x0112, 3})
	_ = binary.Write(&tiff, binary.LittleEndian, uint32(1))
	_ = binary.Write(&tiff, binary.LittleEndian, []uint16{6, 0})
	_ = binary.Write(&tiff, binary.LittleEndian, uint32(0))
	tiff.Write(desc)
	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	return append(seg, payload...)
}

func jpegWithExif(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, jpeg.Encode(&b, gradient(w, h), &jpeg.Options{Quality: 90}))
	raw := b.Bytes()
	return append(append([]byte{0xFF, 0xD8}, exifSegment()...), raw[2:]...)
}

func TestToWebPStripsExifAndAutoRotates(t *testing.T) {
	in := jpegWithExif(t, 400, 200)
	src, err := vips.NewImageFromBuffer(in)
	require.NoError(t, err)
	require.True(t, src.HasExif(), "fixture must carry EXIF")
	src.Close()

	res, err := NewConverter().ToWebP(in, Options{MaxDimension: 2000, Quality: 80})
	require.NoError(t, err)

	out, err := vips.NewImageFromBuffer(res.Data)
	require.NoError(t, err)
	defer out.Close()
	assert.Equal(t, vips.ImageTypeWEBP, out.Format())
	assert.False(t, out.HasExif(), "EXIF must be removed")
	assert.NotContains(t, string(res.Data), "GPS -6.2")
	assert.Equal(t, 200, res.Width, "orientation 6 rotates the image")
	assert.Equal(t, 400, res.Height)
}

func TestToWebPBoundsLongestSide(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, gradient(3000, 1500)))
	res, err := NewConverter().ToWebP(b.Bytes(), Options{MaxDimension: 2000, Quality: 80})
	require.NoError(t, err)
	assert.Equal(t, 2000, res.Width)
	assert.Equal(t, 1000, res.Height)
}

func TestToWebPBuildsOnlySmallerVariants(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, jpeg.Encode(&b, gradient(1200, 600), nil))
	res, err := NewConverter().ToWebP(b.Bytes(), Options{MaxDimension: 2000, Quality: 80, Widths: ResponsiveWidths})
	require.NoError(t, err)
	require.Len(t, res.Variants, 2)
	assert.Equal(t, 480, res.Variants[0].Width)
	assert.Equal(t, 960, res.Variants[1].Width)
	assert.Equal(t, 480, res.Variants[1].Height)
	for _, v := range res.Variants {
		assert.Equal(t, "RIFF", string(v.Data[:4]))
	}
}

func TestToWebPConvertsWebPInput(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, gradient(64, 64)))
	first, err := NewConverter().ToWebP(b.Bytes(), Options{MaxDimension: 2000, Quality: 80})
	require.NoError(t, err)
	second, err := NewConverter().ToWebP(first.Data, Options{MaxDimension: 2000, Quality: 80})
	require.NoError(t, err)
	assert.Equal(t, 64, second.Width)
}

func TestToWebPRejectsGarbage(t *testing.T) {
	_, err := NewConverter().ToWebP([]byte("not an image"), Options{Quality: 80})
	assert.Error(t, err)
}
