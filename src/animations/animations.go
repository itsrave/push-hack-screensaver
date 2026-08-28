// Package animations holds the screensaver's visual animations. Each animation
// is one self-contained value of type Anim, registered in the List below.
//
// ── Adding an animation ──────────────────────────────────────────────────────
// 1. Copy template.go.txt to a new file, e.g. myanim.go.
// 2. Fill in Frame (draws the 960x160 display) and, optionally, Pads / Buttons
//    (the Push LED grid + buttons). Use the shared helpers in util.go:
//    hsv(), fastSin(), NearestIndex(), padRamp(), W, H.
// 3. Add your var to the List slice below. Its position is the animation index
//    (persisted in config), so append to the end to keep existing indexes stable.
// That's it — it shows up in the SAVER tab automatically.
package animations

import "image"

// Anim is one screensaver animation. Only Name and Frame are required; Pads and
// Buttons are optional (nil = that surface stays dark for this animation).
type Anim struct {
	Name string
	// Frame draws one full display frame into img at phase t (seconds, already
	// speed-scaled). img is 960x160 NRGBA, reused between frames — clear what you
	// need to.
	Frame func(img *image.NRGBA, t float64)
	// Pads returns a Push palette index (0 = off) for each of the 64 pads, in
	// note order (36..99). Optional.
	Pads func(t float64) [64]uint8
	// Buttons returns a Push palette index for each of `count` buttons the host
	// wants animated. Optional.
	Buttons func(t float64, count int) []uint8
}

// List is the ordered registry — index == animation number in config/UI.
// Append new animations to the end so existing indexes stay stable.
var List = []Anim{Rainbow, Twinkle, Starfield}

// Names returns the animation names in registry order.
func Names() []string {
	n := make([]string, len(List))
	for i, a := range List {
		n[i] = a.Name
	}
	return n
}
