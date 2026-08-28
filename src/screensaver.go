package main

import (
	"image"
	"log"
	"sync"
	"time"

	"github.com/federico-pepe/ableton-push-hack/core/alsaseq"
	"github.com/federico-pepe/ableton-push-hack/core/pmclient"
	"github.com/federico-pepe/ableton-push-hack/core/push3"
)

// App holds all screensaver state. One instance per process.
type App struct {
	mu           sync.Mutex
	cfg          Config
	cfgPath      string
	active       bool
	lastActivity time.Time
	stopAnim     chan struct{}

	pm       *pmclient.Client
	out      *alsaseq.Client // LED output (nil until MIDI set up)
	pushAddr alsaseq.Addr
}

func newApp(cfgPath string, cfg Config, pmBase string) *App {
	return &App{
		cfg:          cfg,
		cfgPath:      cfgPath,
		lastActivity: time.Now(),
		pm:           pmclient.New(pmBase),
	}
}

// watchIdle is the activation state machine: poll every 500ms, take over the
// display after IdleSeconds of no user input, and hand it back the moment input
// resumes (or the hack is disabled).
func (a *App) watchIdle() {
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for range tick.C {
		a.mu.Lock()
		enabled := a.cfg.Enabled
		idle := time.Duration(a.cfg.IdleSeconds) * time.Second
		since := time.Since(a.lastActivity)
		active := a.active
		a.mu.Unlock()

		switch {
		case active && (!enabled || since < idle):
			a.deactivate()
		case !active && enabled && since >= idle:
			a.activate()
		}
	}
}

func (a *App) activate() {
	a.mu.Lock()
	if a.active {
		a.mu.Unlock()
		return
	}
	a.active = true
	a.stopAnim = make(chan struct{})
	stop := a.stopAnim
	a.mu.Unlock()

	if err := a.pm.SetMode(2); err != nil { // takeover
		log.Printf("activate: SetMode(2): %v", err)
	}
	log.Printf("screensaver: on")
	go a.renderLoop(stop)
	go a.ledLoop(stop)
}

func (a *App) deactivate() {
	a.mu.Lock()
	if !a.active {
		a.mu.Unlock()
		return
	}
	a.active = false
	close(a.stopAnim)
	a.mu.Unlock()

	if err := a.pm.SetMode(0); err != nil { // passthrough
		log.Printf("deactivate: SetMode(0): %v", err)
	}
	log.Printf("screensaver: off")
}

// renderLoop pushes ~20fps animation frames until stop is closed. It reads the
// current animation/speed each frame so panel edits take effect live.
func (a *App) renderLoop(stop chan struct{}) {
	img := image.NewNRGBA(image.Rect(0, 0, push3.VisW, push3.VisH))
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()

	var phase float64
	last := time.Now()
	stars = nil // fresh starfield each activation

	for {
		select {
		case <-stop:
			return
		case now := <-tick.C:
			dt := now.Sub(last).Seconds()
			last = now

			a.mu.Lock()
			anim := a.cfg.Animation
			speed := float64(a.cfg.Speed)
			a.mu.Unlock()

			step := dt * speed * 0.2
			phase += step
			if anim == 2 {
				moveStars(step)
			}
			drawFrame(img, anim, phase)

			if err := a.pm.PushImage(img); err != nil {
				// push-manager/display gone — stop hammering it; idle watcher
				// will retry activation later.
				log.Printf("render: PushImage: %v", err)
				return
			}
		}
	}
}

// ledLoop animates the 64 pad-grid LEDs while the screensaver is active,
// mirroring the screen animation. Runs at ~12fps and only sends pads whose
// palette index changed since the last frame, keeping MIDI traffic modest. On
// stop it clears every pad so no glow is left behind.
func (a *App) ledLoop(stop chan struct{}) {
	a.mu.Lock()
	out := a.out
	dst := a.pushAddr
	a.mu.Unlock()
	if out == nil {
		return // MIDI unavailable — display animation still runs
	}

	var last [64]uint8
	for i := range last {
		last[i] = 255 // invalid velocity → forces every pad to send on frame 1
	}
	defer func() {
		for i := 0; i < 64; i++ {
			out.SendNote(dst, 0, byte(36+i), 0) //nolint:errcheck
		}
	}()

	tick := time.NewTicker(80 * time.Millisecond) // ~12fps
	defer tick.Stop()
	var phase float64
	prev := time.Now()
	for {
		select {
		case <-stop:
			return
		case now := <-tick.C:
			dt := now.Sub(prev).Seconds()
			prev = now
			a.mu.Lock()
			anim := a.cfg.Animation
			speed := float64(a.cfg.Speed)
			a.mu.Unlock()
			phase += dt * speed * 0.2

			cols := padColorsFor(anim, phase)
			for i := 0; i < 64; i++ {
				if cols[i] != last[i] {
					out.SendNote(dst, 0, byte(36+i), cols[i]) //nolint:errcheck
					last[i] = cols[i]
				}
			}
		}
	}
}

// depWatch logs a clear, state-transition-only warning when push-manager or
// push-display isn't reachable — the mandated dependency watcher for a
// display-owning hack.
func (a *App) depWatch() {
	var lastState string
	report := func(s string) {
		if s != lastState {
			log.Print(s)
			lastState = s
		}
	}
	for {
		st, err := a.pm.DisplayStatus()
		switch {
		case err != nil:
			report("dep: push-manager unreachable on :7701 — screensaver can't draw")
		case !st.Connected:
			report("dep: push-manager up but push-display framebuffer not connected")
		default:
			report("dep: display ready")
		}
		time.Sleep(15 * time.Second)
	}
}
