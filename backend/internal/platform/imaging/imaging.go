// Package imaging converts uploaded images to metadata-free WebP with libvips.
package imaging

import (
	"fmt"
	"log/slog"

	"github.com/davidbyttow/govips/v2/vips"
)

// ResponsiveWidths are the CMS image variant widths from PRD Section 7.
var ResponsiveWidths = []int{480, 960, 1600}

// Startup initializes libvips once per process.
func Startup(logger *slog.Logger) error {
	vips.LoggingSettings(func(domain string, level vips.LogLevel, msg string) {
		logger.Warn("libvips", "domain", domain, "level", int(level), "message", msg)
	}, vips.LogLevelWarning)
	if err := vips.Startup(&vips.Config{MaxCacheMem: 64 << 20, MaxCacheSize: 100}); err != nil {
		return fmt.Errorf("start libvips: %w", err)
	}
	return nil
}

// Shutdown releases libvips resources.
func Shutdown() {
	vips.Shutdown()
}

// Options controls the WebP conversion.
type Options struct {
	MaxDimension int
	Quality      int
	Widths       []int
}

// Variant is a resized copy of the main image.
type Variant struct {
	Width  int
	Height int
	Data   []byte
}

// Result is the converted image and its variants.
type Result struct {
	Data     []byte
	Width    int
	Height   int
	Variants []Variant
}

// Converter turns images into WebP.
type Converter struct{}

// NewConverter returns a Converter; Startup must have been called.
func NewConverter() *Converter { return &Converter{} }

// ToWebP auto-rotates, strips metadata, bounds the longest side and encodes WebP plus smaller width variants.
func (c *Converter) ToWebP(data []byte, opts Options) (Result, error) {
	img, err := vips.NewImageFromBuffer(data)
	if err != nil {
		return Result{}, fmt.Errorf("decode image: %w", err)
	}
	defer img.Close()
	if err := img.AutoRotate(); err != nil {
		return Result{}, fmt.Errorf("auto-rotate image: %w", err)
	}
	if err := img.RemoveMetadata(); err != nil {
		return Result{}, fmt.Errorf("strip metadata: %w", err)
	}
	if longest := max(img.Width(), img.Height()); opts.MaxDimension > 0 && longest > opts.MaxDimension {
		if err := img.Resize(float64(opts.MaxDimension)/float64(longest), vips.KernelLanczos3); err != nil {
			return Result{}, fmt.Errorf("resize image: %w", err)
		}
	}
	main, err := export(img, opts.Quality)
	if err != nil {
		return Result{}, err
	}
	res := Result{Data: main, Width: img.Width(), Height: img.Height()}
	for _, w := range opts.Widths {
		if w >= res.Width {
			continue
		}
		v, err := variant(img, w, opts.Quality)
		if err != nil {
			return Result{}, err
		}
		res.Variants = append(res.Variants, v)
	}
	return res, nil
}

func variant(img *vips.ImageRef, width, quality int) (Variant, error) {
	cp, err := img.Copy()
	if err != nil {
		return Variant{}, fmt.Errorf("copy image: %w", err)
	}
	defer cp.Close()
	if err := cp.Resize(float64(width)/float64(cp.Width()), vips.KernelLanczos3); err != nil {
		return Variant{}, fmt.Errorf("resize variant: %w", err)
	}
	data, err := export(cp, quality)
	if err != nil {
		return Variant{}, err
	}
	return Variant{Width: cp.Width(), Height: cp.Height(), Data: data}, nil
}

func export(img *vips.ImageRef, quality int) ([]byte, error) {
	params := vips.NewWebpExportParams()
	params.StripMetadata = true
	params.Quality = quality
	data, _, err := img.ExportWebp(params)
	if err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}
	return data, nil
}
