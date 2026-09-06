package animations

import (
	"image"
	"image/color"
	"math"

	"github.com/federico-pepe/ableton-push-hack/core/gfx/text"
)

// Twinkle: little ASCII stars blinking and cycling colour on the display; the
// pads fade in and out like stars (real dimming via the palette ramp), and the
// buttons blink independently too.
var Twinkle = Anim{
	Name:    "Twinkle",
	Frame:   twinkleFrame,
	Pads:    twinklePads,
	Buttons: twinkleButtons,
}

// ── screen: ASCII stars ──────────────────────────────────────────────────────

type twStar struct {
	x, y, phase float64
	scale       int
}

var twStars []twStar

func twinkleFrame(img *image.NRGBA, t float64) {
	if twStars == nil {
		twStars = make([]twStar, 60)
		for i := range twStars {
			sc := 1
			if rng.Float64() < 0.3 {
				sc = 2
			}
			twStars[i] = twStar{
				x:     rng.Float64() * float64(W-14),
				y:     8 + rng.Float64()*float64(H-16),
				phase: rng.Float64() * 2 * math.Pi,
				scale: sc,
			}
		}
	}
	for i := range img.Pix {
		img.Pix[i] = 0 // black field
	}
	for _, s := range twStars {
		b := (fastSin(s.phase+t*3) + 1) / 2 // 0..1
		if b < 0.25 {
			continue
		}
		ch := "."
		switch {
		case b > 0.75:
			ch = "*"
		case b > 0.5:
			ch = "+"
		}
		hue := math.Mod(s.phase/(2*math.Pi)+t*0.1, 1)
		r, g, bl := NearestRGB(hsv(hue, 1, b))
		text.DrawScaled(img, int(s.x), int(s.y), s.scale, ch, color.NRGBA{r, g, bl, 255})
	}
}

// ── pads: fading twinkle ─────────────────────────────────────────────────────

var (
	padPhase [64]float64
	padHue   [64]int
	padInit  bool
)

func twinklePads(t float64) [64]uint8 {
	if !padInit {
		for i := 0; i < 64; i++ {
			padPhase[i] = rng.Float64() * 2 * math.Pi
			padHue[i] = rng.Intn(RingN)
		}
		padInit = true
	}
	var out [64]uint8
	for i := 0; i < 64; i++ {
		bl := (fastSin(padPhase[i]+t*3) + 1) / 2 // 0..1 smooth
		level := int(bl*float64(Levels-1) + 0.5)
		out[i] = padRamp(padHue[i]+int(t*2), level) // dims/fades; level 0 = off
	}
	return out
}

// ── buttons: independent blink ───────────────────────────────────────────────

var (
	btnPhase []float64
	btnHue   []int
)

func twinkleButtons(t float64, count int) []uint8 {
	if len(btnPhase) != count {
		btnPhase = make([]float64, count)
		btnHue = make([]int, count)
		for i := 0; i < count; i++ {
			btnPhase[i] = rng.Float64() * 2 * math.Pi
			btnHue[i] = rng.Intn(RingN)
		}
	}
	out := make([]uint8, count)
	for i := 0; i < count; i++ {
		if fastSin(btnPhase[i]+t*3) > 0.35 {
			out[i] = padRamp(btnHue[i]+int(t*2), Levels-1) // full-bright blink
		}
	}
	return out
}
