package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/federico-pepe/ableton-push-hack/core/push3"
)

// Animations draw a full frame into a reused 960x160 NRGBA. `t` is a phase in
// (already speed-scaled) seconds; the starfield instead advances via moveStars
// so the render loop controls its motion rate directly.

// sinLUT is a 256-entry sine table — plasma does thousands of lookups per
// frame and we only render while the device is idle, but there's no reason to
// spend a real math.Sin on each. ponytail: LUT, drop it if profiling says the
// Sin cost was noise.
var sinLUT [256]float64

func init() {
	for i := range sinLUT {
		sinLUT[i] = math.Sin(float64(i) / 256 * 2 * math.Pi)
	}
}

func fastSin(x float64) float64 {
	i := int(x/(2*math.Pi)*256) & 255
	return sinLUT[i]
}

// drawFrame renders animation `anim` at phase `t` into img.
func drawFrame(img *image.NRGBA, anim int, t float64) {
	switch anim {
	case 1:
		drawPlasma(img, t)
	case 2:
		drawStarfield(img)
	default:
		drawRainbow(img, t)
	}
}

// drawRainbow: a horizontal hue gradient scrolling left. Per-column hue only,
// so it runs at full resolution cheaply.
func drawRainbow(img *image.NRGBA, t float64) {
	w, h := push3.VisW, push3.VisH
	for x := 0; x < w; x++ {
		hue := math.Mod(float64(x)/float64(w)+t*0.15, 1.0)
		r, g, b := hsv(hue, 1, 1)
		col := color.NRGBA{r, g, b, 255}
		for y := 0; y < h; y++ {
			img.SetNRGBA(x, y, col)
		}
	}
}

// drawPlasma: classic multi-sine plasma, computed at 1/4 resolution and
// block-upscaled 4x so per-pixel sine cost stays small.
func drawPlasma(img *image.NRGBA, t float64) {
	const scale = 4
	w, h := push3.VisW, push3.VisH
	for cy := 0; cy < h; cy += scale {
		for cx := 0; cx < w; cx += scale {
			fx, fy := float64(cx)/24, float64(cy)/24
			v := fastSin(fx+t) +
				fastSin(fy+t*1.3) +
				fastSin((fx+fy)/2+t*0.7) +
				fastSin(math.Sqrt(fx*fx+fy*fy)+t)
			hue := math.Mod(v/4+t*0.05+1, 1.0)
			r, g, b := hsv(hue, 0.9, 1)
			col := color.NRGBA{r, g, b, 255}
			for dy := 0; dy < scale && cy+dy < h; dy++ {
				for dx := 0; dx < scale && cx+dx < w; dx++ {
					img.SetNRGBA(cx+dx, cy+dy, col)
				}
			}
		}
	}
}

type star struct{ x, y, z float64 }

var (
	stars   []star
	starRNG = rand.New(rand.NewSource(1))
)

func newStar() star {
	return star{
		x: starRNG.Float64()*2 - 1,
		y: starRNG.Float64()*2 - 1,
		z: starRNG.Float64()*0.9 + 0.1,
	}
}

// moveStars advances every star toward the viewer by dt (already speed-scaled).
// Called by the render loop before drawStarfield.
func moveStars(dt float64) {
	if stars == nil {
		stars = make([]star, 140)
		for i := range stars {
			stars[i] = newStar()
		}
	}
	for i := range stars {
		stars[i].z -= dt
	}
}

// drawStarfield projects the current stars onto a black field, respawning any
// that flew past the viewer or off-screen.
func drawStarfield(img *image.NRGBA) {
	w, h := push3.VisW, push3.VisH
	for i := range img.Pix {
		img.Pix[i] = 0 // black; alpha 0 is fine, display ignores it
	}
	cx, cy := float64(w)/2, float64(h)/2
	for i := range stars {
		s := &stars[i]
		if s.z <= 0.05 {
			*s = newStar()
			s.z = 1.0
			continue
		}
		px := cx + s.x/s.z*cx
		py := cy + s.y/s.z*cy
		if px < 0 || px >= float64(w) || py < 0 || py >= float64(h) {
			*s = newStar()
			s.z = 1.0
			continue
		}
		b := uint8(255 * (1 - s.z)) // brighter as it nears
		size := 1
		if s.z < 0.4 {
			size = 2
		}
		col := color.NRGBA{b, b, b, 255}
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

// hsv converts h,s,v in [0,1] to 8-bit RGB.
func hsv(h, s, v float64) (uint8, uint8, uint8) {
	h = math.Mod(h, 1)
	if h < 0 {
		h += 1
	}
	i := int(h * 6)
	f := h*6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - f*s)
	tt := v * (1 - (1-f)*s)
	var r, g, b float64
	switch i % 6 {
	case 0:
		r, g, b = v, tt, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, tt
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = tt, p, v
	case 5:
		r, g, b = v, p, q
	}
	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}
