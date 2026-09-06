package animations

import (
	"image"
	"image/color"
)

// Starfield: warp-speed stars flying out from the centre; the pads sparkle white.
var Starfield = Anim{
	Name:  "Starfield",
	Frame: starfieldFrame,
	Pads:  starfieldPads,
}

type star struct{ x, y, z float64 }

var (
	stars    []star
	starLast float64 // last phase, to derive per-frame motion from absolute t
)

func newStar() star {
	return star{x: rng.Float64()*2 - 1, y: rng.Float64()*2 - 1, z: rng.Float64()*0.9 + 0.1}
}

func starfieldFrame(img *image.NRGBA, t float64) {
	if stars == nil {
		stars = make([]star, 140)
		for i := range stars {
			stars[i] = newStar()
		}
		starLast = t
	}
	dt := t - starLast
	starLast = t
	if dt < 0 {
		dt = 0
	}

	for i := range img.Pix {
		img.Pix[i] = 0
	}
	cx, cy := float64(W)/2, float64(H)/2
	for i := range stars {
		s := &stars[i]
		s.z -= dt
		if s.z <= 0.05 {
			*s = newStar()
			s.z = 1.0
			continue
		}
		px := cx + s.x/s.z*cx
		py := cy + s.y/s.z*cy
		if px < 0 || px >= float64(W) || py < 0 || py >= float64(H) {
			*s = newStar()
			s.z = 1.0
			continue
		}
		b := uint8(255 * (1 - s.z))
		size := 1
		if s.z < 0.4 {
			size = 2
		}
		sr, sg, sb := NearestRGB(b, b, b)
		col := color.NRGBA{sr, sg, sb, 255}
		for dy := 0; dy < size; dy++ {
			for dx := 0; dx < size; dx++ {
				setPix(img, int(px)+dx, int(py)+dy, col)
			}
		}
	}
}

func setPix(img *image.NRGBA, x, y int, c color.NRGBA) {
	if x < 0 || y < 0 || x >= img.Rect.Dx() || y >= img.Rect.Dy() {
		return
	}
	img.SetNRGBA(x, y, c)
}

// starfieldPads: sparse white sparkle on the grid.
func starfieldPads(t float64) [64]uint8 {
	var out [64]uint8
	for i := 0; i < 64; i++ {
		if fastSin(float64(i)*0.7+t*2) > 0.7 {
			out[i] = 122 // white
		}
	}
	return out
}
