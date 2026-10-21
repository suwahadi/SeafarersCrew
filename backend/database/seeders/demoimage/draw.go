package demoimage

import (
	"image"
	"image/color"
	"math"
	"slices"
)

func lerp(a, b color.RGBA, t float64) color.RGBA {
	mix := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.RGBA{R: mix(a.R, b.R), G: mix(a.G, b.G), B: mix(a.B, b.B), A: 255}
}

func gradient(img *image.RGBA, r image.Rectangle, top, bottom color.RGBA) {
	r = r.Intersect(img.Bounds())
	height := max(r.Dy()-1, 1)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		c := lerp(top, bottom, float64(y-r.Min.Y)/float64(height))
		for x := r.Min.X; x < r.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func rect(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	r = r.Intersect(img.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func disc(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	ring(img, cx, cy, radius, 0, c)
}

func ring(img *image.RGBA, cx, cy, outer, inner int, c color.RGBA) {
	paintRing(img, cx, cy, outer, inner, cy-outer, c)
}

func lowerRing(img *image.RGBA, cx, cy, outer, inner int, c color.RGBA) {
	paintRing(img, cx, cy, outer, inner, cy, c)
}

func paintRing(img *image.RGBA, cx, cy, outer, inner, fromY int, c color.RGBA) {
	b := img.Bounds()
	o2, i2 := outer*outer, inner*inner
	for y := max(fromY, b.Min.Y); y <= min(cy+outer, b.Max.Y-1); y++ {
		for x := max(cx-outer, b.Min.X); x <= min(cx+outer, b.Max.X-1); x++ {
			d := (x-cx)*(x-cx) + (y-cy)*(y-cy)
			if d <= o2 && d >= i2 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func polygon(img *image.RGBA, pts []image.Point, c color.RGBA) {
	if len(pts) < 3 {
		return
	}
	b := img.Bounds()
	minY, maxY := pts[0].Y, pts[0].Y
	for _, p := range pts {
		minY, maxY = min(minY, p.Y), max(maxY, p.Y)
	}
	for y := max(minY, b.Min.Y); y <= min(maxY, b.Max.Y-1); y++ {
		var xs []int
		for i := range pts {
			a, d := pts[i], pts[(i+1)%len(pts)]
			if (a.Y <= y && d.Y > y) || (d.Y <= y && a.Y > y) {
				t := float64(y-a.Y) / float64(d.Y-a.Y)
				xs = append(xs, a.X+int(math.Round(t*float64(d.X-a.X))))
			}
		}
		slices.Sort(xs)
		for i := 0; i+1 < len(xs); i += 2 {
			for x := max(xs[i], b.Min.X); x <= min(xs[i+1], b.Max.X-1); x++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func wave(img *image.RGBA, base int, amplitude, frequency, phase float64, c color.RGBA) {
	b := img.Bounds()
	for x := b.Min.X; x < b.Max.X; x++ {
		top := base + int(math.Round(amplitude*math.Sin(float64(x)*frequency+phase)))
		for y := max(top, b.Min.Y); y < b.Max.Y; y++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func thickLine(img *image.RGBA, a, b image.Point, width int, c color.RGBA) {
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	length := math.Hypot(dx, dy)
	if length == 0 {
		disc(img, a.X, a.Y, width/2, c)
		return
	}
	nx, ny := int(-dy/length*float64(width)/2), int(dx/length*float64(width)/2)
	polygon(img, []image.Point{
		{X: a.X + nx, Y: a.Y + ny}, {X: b.X + nx, Y: b.Y + ny},
		{X: b.X - nx, Y: b.Y - ny}, {X: a.X - nx, Y: a.Y - ny},
	}, c)
}
