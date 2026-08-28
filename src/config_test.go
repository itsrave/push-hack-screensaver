package main

import "testing"

// TestClamp: a bad/partial config can't wedge the daemon.
func TestClamp(t *testing.T) {
	cases := []struct{ in, want Config }{
		{Config{Animation: 99, IdleSeconds: 0, Speed: 0}, Config{Animation: 0, IdleSeconds: 5, Speed: 1}},
		{Config{Animation: -1, IdleSeconds: 99999, Speed: 50}, Config{Animation: 0, IdleSeconds: 3600, Speed: 10}},
		{Config{Enabled: true, Animation: 2, IdleSeconds: 30, Speed: 5}, Config{Enabled: true, Animation: 2, IdleSeconds: 30, Speed: 5}},
	}
	for i, c := range cases {
		got := c.in
		got.clamp()
		if got != c.want {
			t.Errorf("case %d: clamp(%+v) = %+v, want %+v", i, c.in, got, c.want)
		}
	}
}

