// Package demoimage draws deterministic maritime placeholder images for the demo seed.
package demoimage

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand/v2"
	"strings"
)

// Image sizes in pixels.
const (
	SceneWidth  = 1600
	SceneHeight = 900
	LogoSize    = 400
)

type scene func(img *image.RGBA, p palette, rng *rand.Rand)

var scenes = []scene{horizonScene, shipScene, compassScene, wavesScene, anchorScene}

// SceneCount is the number of distinct scene themes.
func SceneCount() int { return len(scenes) }

// Scene returns a 1600x900 PNG whose theme and colors follow index; the same index always yields the same bytes.
func Scene(index int) ([]byte, error) {
	if index < 0 {
		return nil, fmt.Errorf("scene index %d is negative", index)
	}
	rng := rand.New(rand.NewPCG(uint64(index)+1, 0x5EA))
	p := palettes[(index+index/len(scenes))%len(palettes)]
	img := image.NewRGBA(image.Rect(0, 0, SceneWidth, SceneHeight))
	scenes[index%len(scenes)](img, p, rng)
	return encode(img)
}

// Logo returns a 400x400 PNG with a colored emblem and the first two letters of name as blocky initials.
func Logo(index int, name string) ([]byte, error) {
	if index < 0 {
		return nil, fmt.Errorf("logo index %d is negative", index)
	}
	bg := logoColors[index%len(logoColors)]
	soft := lerp(bg, rgb(255, 255, 255), 0.2)
	img := image.NewRGBA(image.Rect(0, 0, LogoSize, LogoSize))
	rect(img, img.Bounds(), bg)
	half := LogoSize / 2
	switch (index / len(logoColors)) % 4 {
	case 0:
		disc(img, half, half, 180, soft)
	case 1:
		polygon(img, []image.Point{{X: half, Y: 20}, {X: 380, Y: half}, {X: half, Y: 380}, {X: 20, Y: half}}, soft)
	case 2:
		rect(img, image.Rect(30, 30, 370, 370), soft)
	default:
		wave(img, 280, 24, 0.045, float64(index), soft)
	}
	drawInitials(img, initials(name), rgb(255, 255, 255))
	return encode(img)
}

func initials(name string) string {
	var out, rest []rune
	for _, word := range strings.Fields(strings.ToUpper(name)) {
		var letters []rune
		for _, r := range word {
			if _, ok := glyphs[r]; ok {
				letters = append(letters, r)
			}
		}
		if len(letters) == 0 {
			continue
		}
		if len(out) == 0 {
			rest = letters[1:]
		}
		out = append(out, letters[0])
		if len(out) == 2 {
			break
		}
	}
	if len(out) == 1 && len(rest) > 0 {
		out = append(out, rest[0])
	}
	return string(out)
}

func drawInitials(img *image.RGBA, text string, c color.RGBA) {
	const cell, gap = 22, 1
	letters := []rune(text)
	if len(letters) == 0 {
		return
	}
	letterWidth := glyphWidth * cell
	total := len(letters)*letterWidth + (len(letters)-1)*cell
	x0 := (LogoSize - total) / 2
	y0 := (LogoSize - glyphHeight*cell) / 2
	for i, r := range letters {
		gx := x0 + i*(letterWidth+cell)
		for row, line := range glyphs[r] {
			for col, ch := range line {
				if ch == '#' {
					rect(img, image.Rect(gx+col*cell, y0+row*cell, gx+(col+1)*cell-gap, y0+(row+1)*cell-gap), c)
				}
			}
		}
	}
}

func encode(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return buf.Bytes(), nil
}

func between(rng *rand.Rand, lo, hi int) int { return lo + rng.IntN(hi-lo+1) }

func skyAndSea(img *image.RGBA, p palette, horizon int) {
	gradient(img, image.Rect(0, 0, SceneWidth, horizon), p.skyTop, p.skyBottom)
	gradient(img, image.Rect(0, horizon, SceneWidth, SceneHeight), p.seaTop, p.seaBottom)
}

func swell(img *image.RGBA, p palette, horizon int, rng *rand.Rand, layers int) {
	for i := range layers {
		t := float64(i+1) / float64(layers+1)
		c := lerp(p.seaTop, p.seaBottom, t)
		base := horizon + 40 + i*(SceneHeight-horizon-40)/layers
		wave(img, base, float64(6+i*5), 0.006+rng.Float64()*0.012, rng.Float64()*math.Pi*2, c)
	}
}

func horizonScene(img *image.RGBA, p palette, rng *rand.Rand) {
	horizon := between(rng, 480, 560)
	skyAndSea(img, p, horizon)
	sunX, sunY, radius := between(rng, 300, 1300), horizon-between(rng, 40, 140), between(rng, 80, 130)
	ring(img, sunX, sunY, radius+40, radius+20, lerp(p.skyBottom, p.accent, 0.35))
	disc(img, sunX, sunY, radius, p.accent)
	for i := range 9 {
		w := 260 - i*26
		y := horizon + 14 + i*22
		rect(img, image.Rect(sunX-w/2, y, sunX+w/2, y+6), lerp(p.seaTop, p.accent, 0.55))
	}
	swell(img, p, horizon, rng, 4)
}

func shipScene(img *image.RGBA, p palette, rng *rand.Rand) {
	horizon := between(rng, 470, 530)
	skyAndSea(img, p, horizon)
	disc(img, between(rng, 200, 1400), between(rng, 120, 260), between(rng, 40, 70), lerp(p.skyTop, p.accent, 0.7))
	water := horizon + 90
	x0, length, depth := between(rng, 220, 620), between(rng, 680, 820), 90
	hull := []image.Point{
		{X: x0, Y: water - depth}, {X: x0 + length, Y: water - depth},
		{X: x0 + length - 120, Y: water}, {X: x0 + 60, Y: water},
	}
	polygon(img, hull, p.dark)
	rect(img, image.Rect(x0+60, water-depth-14, x0+length-40, water-depth), lerp(p.dark, p.accent, 0.25))
	house := image.Rect(x0+80, water-depth-130, x0+270, water-depth-14)
	rect(img, house, lerp(p.dark, rgb(255, 255, 255), 0.18))
	rect(img, image.Rect(house.Min.X+20, house.Min.Y-50, house.Max.X-20, house.Min.Y), lerp(p.dark, rgb(255, 255, 255), 0.12))
	rect(img, image.Rect(house.Min.X+60, house.Min.Y-110, house.Min.X+100, house.Min.Y-50), p.accent)
	for i := range 4 {
		rect(img, image.Rect(house.Min.X+14+i*44, house.Min.Y+30, house.Min.X+40+i*44, house.Min.Y+50), p.accent)
	}
	thickLine(img, image.Pt(x0+length-260, water-depth-14), image.Pt(x0+length-260, water-depth-210), 10, p.dark)
	thickLine(img, image.Pt(x0+length-260, water-depth-210), image.Pt(x0+length-90, water-depth-60), 8, p.dark)
	swell(img, p, horizon+60, rng, 3)
}

func compassScene(img *image.RGBA, p palette, rng *rand.Rand) {
	gradient(img, img.Bounds(), p.skyTop, p.seaBottom)
	cx, cy := SceneWidth/2+between(rng, -60, 60), SceneHeight/2
	ring(img, cx, cy, 360, 340, p.accent)
	ring(img, cx, cy, 300, 292, lerp(p.accent, p.skyTop, 0.4))
	for i := range 36 {
		a := float64(i) * math.Pi / 18
		long := 36
		if i%9 != 0 {
			long = 16
		}
		from := image.Pt(cx+int(math.Cos(a)*340), cy+int(math.Sin(a)*340))
		to := image.Pt(cx+int(math.Cos(a)*float64(340-long)), cy+int(math.Sin(a)*float64(340-long)))
		thickLine(img, from, to, 6, p.accent)
	}
	diagonal := lerp(p.accent, p.skyTop, 0.55)
	for _, a := range []float64{math.Pi / 4, 3 * math.Pi / 4, 5 * math.Pi / 4, 7 * math.Pi / 4} {
		tip := image.Pt(cx+int(math.Cos(a)*200), cy+int(math.Sin(a)*200))
		l := image.Pt(cx+int(math.Cos(a+math.Pi/2)*28), cy+int(math.Sin(a+math.Pi/2)*28))
		r := image.Pt(cx+int(math.Cos(a-math.Pi/2)*28), cy+int(math.Sin(a-math.Pi/2)*28))
		polygon(img, []image.Point{tip, l, image.Pt(cx, cy), r}, diagonal)
	}
	cardinal := []struct {
		reach int
		dx    int
		dy    int
	}{{270, 0, -1}, {270, 0, 1}, {230, 1, 0}, {230, -1, 0}}
	for _, c := range cardinal {
		tip := image.Pt(cx+c.dx*c.reach, cy+c.dy*c.reach)
		l := image.Pt(cx+c.dy*40, cy+c.dx*40)
		r := image.Pt(cx-c.dy*40, cy-c.dx*40)
		polygon(img, []image.Point{tip, l, image.Pt(cx, cy), r}, p.accent)
	}
	disc(img, cx, cy, 34, p.dark)
	disc(img, cx, cy, 16, p.accent)
}

func wavesScene(img *image.RGBA, p palette, rng *rand.Rand) {
	gradient(img, img.Bounds(), p.skyTop, p.skyBottom)
	disc(img, between(rng, 250, 1350), between(rng, 150, 300), between(rng, 70, 120), p.accent)
	const layers = 7
	for i := range layers {
		t := float64(i) / float64(layers-1)
		c := lerp(p.seaTop, p.dark, t)
		base := 360 + i*85
		wave(img, base, float64(18+i*6), 0.004+rng.Float64()*0.008+float64(i)*0.0007, rng.Float64()*math.Pi*2, c)
	}
}

func anchorScene(img *image.RGBA, p palette, rng *rand.Rand) {
	gradient(img, img.Bounds(), p.skyTop, p.seaBottom)
	cx := SceneWidth/2 + between(rng, -120, 120)
	ring(img, cx, 220, 56, 34, p.accent)
	thickLine(img, image.Pt(cx, 276), image.Pt(cx, 740), 38, p.accent)
	thickLine(img, image.Pt(cx-110, 330), image.Pt(cx+110, 330), 32, p.accent)
	lowerRing(img, cx, 560, 220, 188, p.accent)
	polygon(img, []image.Point{{X: cx - 220, Y: 566}, {X: cx - 150, Y: 566}, {X: cx - 250, Y: 480}}, p.accent)
	polygon(img, []image.Point{{X: cx + 150, Y: 566}, {X: cx + 220, Y: 566}, {X: cx + 250, Y: 480}}, p.accent)
	wave(img, 790, 14, 0.01+rng.Float64()*0.01, rng.Float64()*math.Pi*2, p.dark)
}
