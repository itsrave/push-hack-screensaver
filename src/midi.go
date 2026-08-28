package main

import (
	"log"
	"sync"
	"time"

	"github.com/federico-pepe/ableton-push-hack/core/alsaseq"
	"github.com/federico-pepe/ableton-push-hack/core/push3"
)

const push3PortName = "Ableton Push 3 Live Port"

// setupMIDI opens two ALSA seq clients — one to read Push input (idle
// detection) and one to send LEDs — after the boot-settle window. Mirrors
// push-manager's port setup so the two coexist (both just subscribe/send to
// the same RW Push port). Returns the LED-output client and the Push address,
// or nils if the Push port isn't present (daemon still runs, just no LEDs/idle).
func (a *App) setupMIDI() {
	alsaseq.WaitForBootSettle() // USB-A safety: no /dev/snd before uptime >= 30s

	port, ok := alsaseq.FindByName(push3PortName, 0)
	if !ok {
		log.Printf("midi: %q not found — idle detection + LEDs disabled", push3PortName)
		return
	}
	a.pushAddr = port.Addr

	// Output client for LED sends (CapRead = can emit events).
	out, err := alsaseq.Open()
	if err != nil {
		log.Printf("midi: open out: %v", err)
		return
	}
	if _, err := out.CreatePort("Push Hack Screensaver", alsaseq.CapRead|alsaseq.CapSubsRead, alsaseq.PortTypeMidi|alsaseq.PortTypeApp); err != nil {
		log.Printf("midi: create out port: %v", err)
		return
	}
	a.mu.Lock()
	a.out = out
	a.mu.Unlock()

	// Input client subscribed to Push for idle detection.
	in, err := alsaseq.Open()
	if err != nil {
		log.Printf("midi: open in: %v", err)
		return
	}
	if _, err := in.CreatePort("Push Hack Screensaver In", alsaseq.CapWrite|alsaseq.CapSubsWrite, alsaseq.PortTypeMidi|alsaseq.PortTypeApp); err != nil {
		log.Printf("midi: create in port: %v", err)
		return
	}
	if err := in.Subscribe(port.Addr); err != nil {
		log.Printf("midi: subscribe %v: %v", port.Addr, err)
		return
	}

	log.Printf("midi: watching %q (%d:%d) for idle", push3PortName, port.Addr.Client, port.Addr.Port)
	go func() {
		if err := in.ReadLoop(a); err != nil {
			log.Printf("midi: read loop ended: %v", err)
		}
	}()
}

// Fixed/VarLen implement alsaseq.Handler. Only real user-input events reset the
// idle timer — clock/start/stop/sensing are ignored so a running transport (or
// a device that streams continuous clock) doesn't keep the screensaver awake.
func (a *App) Fixed(evType uint8, _ alsaseq.Addr, _ []byte) {
	switch evType {
	case alsaseq.EvNoteOn, alsaseq.EvNoteOff, alsaseq.EvKeyPress,
		alsaseq.EvController, alsaseq.EvChanPress, alsaseq.EvPitchBend:
		a.touch()
	}
}

func (a *App) VarLen(_ uint8, _ alsaseq.Addr, _ []byte) {
	// SysEx etc. — not user input for idle purposes; ignore.
}

// touch marks the device as just-used.
func (a *App) touch() {
	a.mu.Lock()
	a.lastActivity = time.Now()
	a.mu.Unlock()
}

// rainbowLEDs are palette names spanning the hue wheel; rainbowRing resolves
// them once to Push palette velocity indices, shared by the startup sweep and
// the live pad animation.
var rainbowLEDs = []string{"red", "orange", "yellow", "lime", "green", "teal", "sky", "blue", "indigo", "violet", "purple", "pink"}

var (
	ringOnce sync.Once
	ring     []uint8
)

func rainbowRing() []uint8 {
	ringOnce.Do(func() {
		for _, n := range rainbowLEDs {
			if v, ok := push3.ColorByName(n); ok {
				ring = append(ring, v)
			}
		}
		if len(ring) == 0 {
			ring = []uint8{122} // white fallback
		}
	})
	return ring
}

// ledStartupSequence blinks every pad and the main buttons on in a rainbow
// wipe, then clears them — the "all LEDs blink in sequence" flourish on boot.
// Runs once; push-manager re-asserts its own LED state on the next event.
func (a *App) ledStartupSequence() {
	a.mu.Lock()
	out := a.out
	dst := a.pushAddr
	a.mu.Unlock()
	if out == nil {
		return
	}

	colors := rainbowRing()
	colorAt := func(i int) uint8 { return colors[i%len(colors)] }

	// Pad grid: notes 36..99, wiped in order. On pads the Note velocity is the
	// palette index (same 128-entry table as button CC values).
	for i, note := 0, 36; note <= 99; i, note = i+1, note+1 {
		out.SendNote(dst, 0, byte(note), colorAt(i)) //nolint:errcheck
		time.Sleep(12 * time.Millisecond)
	}
	// Buttons: sweep the prominent CCs.
	for i, cc := range sweepButtonCCs {
		out.SendCC(dst, 0, cc, int32(colorAt(i))) //nolint:errcheck
		time.Sleep(12 * time.Millisecond)
	}

	time.Sleep(250 * time.Millisecond)

	// All off.
	for note := 36; note <= 99; note++ {
		out.SendNote(dst, 0, byte(note), 0) //nolint:errcheck
	}
	for _, cc := range sweepButtonCCs {
		out.SendCC(dst, 0, cc, 0) //nolint:errcheck
	}
}

// sweepButtonCCs is a broad set of Push 3 button CCs to include in the startup
// wipe — the screen rows, transport, nav and mode buttons. Not exhaustive, but
// reads as "all the buttons lit".
var sweepButtonCCs = []byte{
	push3.CCScreenTop1, push3.CCScreenTop2, push3.CCScreenTop3, push3.CCScreenTop4,
	push3.CCScreenTop5, push3.CCScreenTop6, push3.CCScreenTop7, push3.CCScreenTop8,
	push3.CCScreenBot1, push3.CCScreenBot2, push3.CCScreenBot3, push3.CCScreenBot4,
	push3.CCScreenBot5, push3.CCScreenBot6, push3.CCScreenBot7, push3.CCScreenBot8,
	push3.CCScene14, push3.CCScene14t, push3.CCScene18, push3.CCScene18t,
	push3.CCScene116, push3.CCScene116t, push3.CCScene132, push3.CCScene132t,
	push3.CCPlay, push3.CCRecord, push3.CCNew, push3.CCDuplicate, push3.CCAutomate,
	push3.CCFixedLength, push3.CCQuantize, push3.CCMetronome, push3.CCTapTempo,
	push3.CCMute, push3.CCSolo, push3.CCStopClips, push3.CCUndo, push3.CCDelete,
	push3.CCDeviceView, push3.CCMixerView, push3.CCClipView, push3.CCSessionView,
	push3.CCRepeat, push3.CCAccent, push3.CCScale, push3.CCLayout, push3.CCNote, push3.CCSession,
	push3.CCOctaveUp, push3.CCOctaveDown, push3.CCPageLeft, push3.CCPageRight,
	push3.CCShift, push3.CCSelect, push3.CCAdd, push3.CCSave, push3.CCSet, push3.CCUserMode,
}
