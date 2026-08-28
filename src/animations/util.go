package animations

import (
	"math"
	"math/rand"
	"sync"

	"github.com/federico-pepe/ableton-push-hack/core/push3"
)

// Display geometry, re-exported for animation authors.
const (
	W = push3.VisW
	H = push3.VisH
)

// rng seeds all animation randomness (star positions, blink phases). Fixed
// seed → reproducible layouts.
var rng = rand.New(rand.NewSource(1))

// ── fast sine ────────────────────────────────────────────────────────────────

var sinLUT [256]float64

func init() {
	for i := range sinLUT {
		sinLUT[i] = math.Sin(float64(i) / 256 * 2 * math.Pi)
	}
}

// fastSin is a 256-entry LUT sine — cheap enough to call per-pixel.
func fastSin(x float64) float64 {
	return sinLUT[int(x/(2*math.Pi)*256)&255]
}

// ── colour ───────────────────────────────────────────────────────────────────

// hsv converts h,s,v in [0,1] to 8-bit RGB.
func hsv(h, s, v float64) (uint8, uint8, uint8) {
	h = math.Mod(h, 1)
	if h < 0 {
		h++
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

// NearestIndex returns the Push palette index whose RGB is closest to (r,g,b).
// This is how an arbitrary animation colour maps onto the hardware's fixed
// 128-colour LED palette — and, crucially, how a dimmed colour lands on a
// darker palette entry (the dk_* shades), giving pads a real fade.
func NearestIndex(r, g, b uint8) uint8 {
	best := uint8(0)
	bestD := 1 << 30
	for _, e := range push3.Palette {
		dr := int(e.RGB.R) - int(r)
		dg := int(e.RGB.G) - int(g)
		db := int(e.RGB.B) - int(b)
		d := dr*dr + dg*dg + db*db
		if d < bestD {
			bestD = d
			best = e.Index
		}
	}
	return best
}

// ── dim ramp ─────────────────────────────────────────────────────────────────

// RingN evenly-spaced hues, each with Levels brightness steps (0 = off), mapped
// to the nearest palette index. padRamp(hue, level) is the LED equivalent of
// hsv(hue, 1, level) — used for the fading twinkle. Built once.
const (
	RingN  = 12
	Levels = 8
)

var (
	ramp     [RingN][Levels]uint8
	rampOnce sync.Once
)

func buildRamp() {
	for h := 0; h < RingN; h++ {
		for l := 0; l < Levels; l++ {
			r, g, b := hsv(float64(h)/RingN, 1, float64(l)/(Levels-1))
			ramp[h][l] = NearestIndex(r, g, b)
		}
	}
}

// padRamp returns the palette index for hue bucket h (0..RingN-1) at brightness
// level (0..Levels-1). level 0 is off.
func padRamp(h, level int) uint8 {
	rampOnce.Do(buildRamp)
	if level <= 0 {
		return 0
	}
	if level >= Levels {
		level = Levels - 1
	}
	return ramp[((h%RingN)+RingN)%RingN][level]
}
