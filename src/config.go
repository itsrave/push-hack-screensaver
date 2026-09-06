package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"screensaver/animations"
)

// animNames indexes the animations, sourced from the animations registry.
// Order is the wire contract with the push-manager SCREENSAVER panel — the
// registry appends new animations, keeping existing indexes stable.
var animNames = animations.Names()

// Config is the user-tunable state, persisted next to hack.json as
// screensaver.json. Kept deliberately small; the Shadow UI panel edits it.
type Config struct {
	Enabled      bool `json:"enabled"`
	Animation    int  `json:"animation"`     // index into animNames
	IdleSeconds  int  `json:"idle_seconds"`  // idle timeout before takeover
	Speed        int  `json:"speed"`         // 1..10 animation speed
	StartupSweep bool `json:"startup_sweep"` // rainbow LED wipe on boot (off by default)
}

func defaultConfig() Config {
	// StartupSweep defaults to false (zero value).
	return Config{Enabled: true, Animation: 1, IdleSeconds: 30, Speed: 5} // 1 = Twinkle
}

// clamp keeps a loaded/patched config inside sane ranges so a bad write can't
// wedge the daemon (e.g. IdleSeconds=0 → constant takeover).
func (c *Config) clamp() {
	if c.Animation < 0 || c.Animation >= len(animNames) {
		c.Animation = 0
	}
	if c.IdleSeconds < 5 {
		c.IdleSeconds = 5
	}
	if c.IdleSeconds > 3600 {
		c.IdleSeconds = 3600
	}
	if c.Speed < 1 {
		c.Speed = 1
	}
	if c.Speed > 10 {
		c.Speed = 10
	}
}

// configPath is screensaver.json in the same dir as the hack.json we were
// started with.
func configPath(hackJSON string) string {
	return filepath.Join(filepath.Dir(hackJSON), "screensaver.json")
}

func loadConfig(path string) Config {
	c := defaultConfig()
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &c) // defaults survive a partial/garbage file
	}
	c.clamp()
	return c
}

// saveConfig writes atomically (tmp + rename), matching the other hacks.
func saveConfig(path string, c Config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// saveMu serialises config writes (config is otherwise guarded by App.mu; this
// just keeps the tmp/rename off the hot path lock).
var saveMu sync.Mutex
