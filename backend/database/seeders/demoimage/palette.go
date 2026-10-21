package demoimage

import "image/color"

type palette struct {
	skyTop, skyBottom color.RGBA
	seaTop, seaBottom color.RGBA
	accent, dark      color.RGBA
}

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{R: r, G: g, B: b, A: 255} }

var palettes = []palette{
	{rgb(11, 37, 69), rgb(244, 162, 97), rgb(19, 76, 120), rgb(8, 28, 52), rgb(255, 214, 102), rgb(6, 18, 36)},
	{rgb(28, 78, 128), rgb(173, 216, 230), rgb(20, 112, 140), rgb(10, 52, 80), rgb(255, 255, 255), rgb(12, 36, 60)},
	{rgb(44, 24, 78), rgb(236, 120, 130), rgb(60, 52, 120), rgb(22, 18, 56), rgb(255, 196, 140), rgb(16, 12, 40)},
	{rgb(18, 80, 82), rgb(190, 232, 210), rgb(16, 124, 120), rgb(8, 58, 62), rgb(255, 240, 180), rgb(6, 36, 40)},
	{rgb(70, 82, 96), rgb(214, 220, 228), rgb(54, 86, 112), rgb(26, 44, 62), rgb(240, 120, 60), rgb(18, 28, 40)},
	{rgb(120, 36, 40), rgb(250, 200, 150), rgb(40, 70, 110), rgb(16, 30, 56), rgb(255, 232, 160), rgb(14, 22, 40)},
}

var logoColors = []color.RGBA{
	rgb(11, 37, 69), rgb(19, 168, 158), rgb(180, 54, 54), rgb(224, 140, 30),
	rgb(86, 62, 150), rgb(30, 110, 60), rgb(40, 90, 160), rgb(70, 74, 82),
}
