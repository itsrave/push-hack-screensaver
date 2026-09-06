# push-hack-screensaver

An idle **screensaver** hack for the Ableton Push 3, built on the
[`push-hack`](https://github.com/federico-pepe/ableton-push-hack) framework.

After a configurable idle period with no pad/button/encoder input, it takes
over the Push 3 display with a full-screen animation. Any input wakes it
instantly. It can also run a rainbow **LED sweep** across every pad and button
on boot (off by default; enable it in the browser config).

## Animations

| # | Name | |
|---|------|--|
| 0 | Rainbow  | scrolling hue gradient |
| 1 | Twinkle *(default)* | little ASCII stars (`* + .`) blinking and cycling colour |
| 2 | Starfield| warp-speed stars |

While active, the 8×8 **pad LEDs** animate to match the selected animation
(rainbow wipe / fading twinkle / white sparkle). In **Twinkle** mode the pads
**fade** in and out (real dimming via the palette's dark shades), and the
function / top / transport **buttons** blink independently too, like the
on-screen stars. All LEDs are cleared when the screensaver exits.

## Adding an animation

Animations live in [`src/animations/`](src/animations/) and are trivial to add —
each is one `Anim` value in the registry:

1. Copy `src/animations/template.go.txt` to `src/animations/myanim.go`.
2. Implement `Frame` (draws the 960×160 display) and optionally `Pads` /
   `Buttons` (the Push LEDs). Shared helpers — `hsv`, `fastSin`, `NearestIndex`,
   `padRamp`, `W`, `H` — are in `util.go`.
3. Add your var to `List` in `animations.go` (append to keep existing indexes
   stable). It appears in the browser config automatically.

No other file needs touching — `Names()`/`List` drive the config, UI and LED loop.

## Configuration — browser

Configured from any browser on the same network at **`http://push.local:7706/`**
(push-manager also links to it via the `web_ui` field in `hack.json`):

- **Enabled** — on/off
- **Animation** — Rainbow / Twinkle / Starfield
- **Idle** — seconds before takeover (5–3600)
- **Speed** — 1–10
- **Startup LED sweep** — rainbow wipe across the pads on boot (default **off**;
  applies on next boot)

Settings persist to `screensaver.json` next to the binary and take effect live
(the startup sweep on next boot); the page also shows whether the screensaver is
currently active.

## Requires (hard dependency)

This is a **display-owning hack**: it draws by calling push-manager's
`/api/display/*` HTTP API (never the shm framebuffer directly — see the
framework's "Display-owning hacks" rule). So it needs, on the same device:

- **push-manager** running (port 7701) — draws frames via its display API.
- **push-display** installed — the LD_PRELOAD hook that puts frames on screen.

The daemon logs a clear warning if either is missing (`dep: …`).

## Build & deploy

```bash
make            # builds build/screensaver (linux/amd64)
./deploy.sh     # scp + install sysvinit service on push.local (PUSH_HOST=... to override)
```

`go.mod` uses a local `replace` to the monorepo's `core/` for development. To
publish through the Push Hack Catalogue, drop the replace and pin the
`core/vX.Y.Z` tag (see the framework's `catalogue/PUBLISHING.md`).

## Notes / known corners

- Screensaver and push-manager's own Shadow UI both draw via display takeover;
  they're mutually exclusive in practice (any input that opens the Shadow UI
  also counts as activity and keeps the screensaver off). No explicit
  coordination beyond that.
- MIDI **clock/start/stop** are ignored for idle detection — only real user
  input (notes/CC/aftertouch/pitch-bend) resets the timer, so a running
  transport doesn't keep the screensaver awake.
- The startup LED sweep runs once; push-manager re-asserts its own LED state on
  the next control event.
