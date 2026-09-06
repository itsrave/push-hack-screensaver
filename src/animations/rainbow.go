package animations

import (
	"image"
	"image/color"
	"math"
)

// Rainbow: a horizontal hue gradient scrolling across the screen; the pad grid
// runs a diagonal rainbow wipe.
var Rainbow = Anim{
	Name: "Rainbow",
	Frame: func(img *image.NRGBA, t float64) {
		for x := 0; x < W; x++ {
			hue := math.Mod(float64(x)/float64(W)+t*0.15, 1)
			r, g, b := NearestRGB(hsv(hue, 1, 1))
			col := color.NRGBA{r, g, b, 255}
			for y := 0; y < H; y++ {
				img.SetNRGBA(x, y, col)
			}
		}
	},
	Pads: func(t float64) [64]uint8 {
		var out [64]uint8
		for i := 0; i < 64; i++ {
			row, col := i/8, i%8
			hue := math.Mod(float64(col+row)/14+t*0.15, 1)
			out[i] = padRamp(int(hue*RingN), Levels-1)
		}
		return out
	},
}
