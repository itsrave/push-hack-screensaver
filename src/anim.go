package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/federico-pepe/ableton-push-hack/core/gfx/text"
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
		drawTwinkle(img, t)
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

// twStar is one ASCII star: a fixed position, a personal blink/hue phase, and
// a draw scale (1 = little, 2 = bigger/nearer).
type twStar struct {
	x, y, phase float64
	scale       int
}

var twStars []twStar

// drawTwinkle: little ASCII stars scattered on a black field, each blinking on
// and off and cycling colour. The glyph shifts with brightness (`*` bright,
// `+` mid, `.` dim) so a star reads as fading in and out, not just toggling.
func drawTwinkle(img *image.NRGBA, t float64) {
	w, h := push3.VisW, push3.VisH
	if twStars == nil {
		twStars = make([]twStar, 60)
		for i := range twStars {
			sc := 1
			if starRNG.Float64() < 0.3 {
				sc = 2
			}
			twStars[i] = twStar{
				x:     starRNG.Float64() * float64(w-14),
				y:     8 + starRNG.Float64()*float64(h-16),
				phase: starRNG.Float64() * 2 * math.Pi,
				scale: sc,
			}
		}
	}
	for i := range img.Pix {
		img.Pix[i] = 0 // black field
	}
	for _, s := range twStars {
		b := (fastSin(s.phase+t*3) + 1) / 2 // 0..1 blink
		if b < 0.25 {
			continue // fully off part of the blink
		}
		var ch string
		switch {
		case b > 0.75:
			ch = "*"
		case b > 0.5:
			ch = "+"
		default:
			ch = "."
		}
		hue := math.Mod(s.phase/(2*math.Pi)+t*0.1, 1) // each star its own, drifting
		r, g, bl := hsv(hue, 1, b)                     // brightness tracks the blink
		text.DrawScaled(img, int(s.x), int(s.y), s.scale, ch, color.NRGBA{r, g, bl, 255})
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

// padColorsFor returns a 64-entry pad LED palette in note order (36..99) for
// the given animation and phase — the pad grid "glow" that plays alongside the
// screen animation while the screensaver is active. Values are Push palette
// indices (0 = off).
func padColorsFor(anim int, t float64) [64]uint8 {
	ring := rainbowRing()
	n := len(ring)
	ensureTwinklePhases(n)
	var out [64]uint8
	for i := 0; i < 64; i++ {
		row, col := i/8, i%8
		switch anim {
		case 2: // starfield: sparse white twinkle on a dark grid
			if fastSin(float64(i)*0.7+t*2) > 0.7 {
				out[i] = 122 // white
			}
		case 1: // twinkle: each pad blinks independently like a star, colour cycling
			if fastSin(padPhase[i]+t*3) > 0.35 {
				out[i] = ring[(padSeed[i]+int(t*2))%n]
			}
		default: // rainbow diagonal wipe
			hue := math.Mod(float64(col+row)/14+t*0.15, 1)
			if hue < 0 {
				hue++
			}
			out[i] = ring[int(hue*float64(n))%n]
		}
	}
	return out
}

// Twinkle LED state: a random phase + colour seed per pad and per twinkle
// button, so they blink out of step with each other (scattered, star-like)
// instead of in a wave.
var (
	padPhase     [64]float64
	padSeed      [64]int
	btnPhase     []float64
	btnSeed      []int
	twinkleReady bool
)

func ensureTwinklePhases(n int) {
	if twinkleReady {
		return
	}
	for i := 0; i < 64; i++ {
		padPhase[i] = starRNG.Float64() * 2 * math.Pi
		padSeed[i] = starRNG.Intn(n)
	}
	btnPhase = make([]float64, len(twinkleButtons))
	btnSeed = make([]int, len(twinkleButtons))
	for i := range twinkleButtons {
		btnPhase[i] = starRNG.Float64() * 2 * math.Pi
		btnSeed[i] = starRNG.Intn(n)
	}
	twinkleReady = true
}

// buttonTwinkleColors returns a palette index per twinkleButtons entry — the
// same independent blink applied to the function/top/transport buttons.
func buttonTwinkleColors(t float64) []uint8 {
	ring := rainbowRing()
	n := len(ring)
	ensureTwinklePhases(n)
	out := make([]uint8, len(twinkleButtons))
	for i := range twinkleButtons {
		if fastSin(btnPhase[i]+t*3) > 0.35 {
			out[i] = ring[(btnSeed[i]+int(t*2))%n]
		}
	}
	return out
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
