package animations

import "testing"

// TestPadRampDims: the dim ramp must actually dim — level 0 is off, and
// brightness increases monotonically enough that the top level is a real, lit
// colour distinct from off.
func TestPadRampDims(t *testing.T) {
	for h := 0; h < RingN; h++ {
		if got := padRamp(h, 0); got != 0 {
			t.Errorf("padRamp(%d,0) = %d, want 0 (off)", h, got)
		}
		if got := padRamp(h, Levels-1); got == 0 {
			t.Errorf("padRamp(%d,%d) = 0, want a lit colour", h, Levels-1)
		}
	}
}

// TestHSVPrimaries: hue wheel endpoints land on the expected primaries.
func TestHSVPrimaries(t *testing.T) {
	if r, g, b := hsv(0, 1, 1); r != 255 || g != 0 || b != 0 {
		t.Errorf("hsv(0) = %d,%d,%d, want 255,0,0", r, g, b)
	}
	if r, g, b := hsv(1.0/3, 1, 1); g != 255 || r != 0 || b != 0 {
		t.Errorf("hsv(1/3) = %d,%d,%d, want 0,255,0", r, g, b)
	}
}

// TestRegistry: List and Names stay in sync and non-empty.
func TestRegistry(t *testing.T) {
	if len(List) == 0 || len(Names()) != len(List) {
		t.Fatalf("registry/names mismatch: %d vs %d", len(List), len(Names()))
	}
	for i, a := range List {
		if a.Name == "" || a.Frame == nil {
			t.Errorf("animation %d missing Name or Frame", i)
		}
	}
}
