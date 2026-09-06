package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/federico-pepe/ableton-push-hack/core/hackcfg"
)

func main() {
	cfgFlag := flag.String("config", "hack.json", "path to hack.json")
	pmBase := flag.String("push-manager", "http://localhost:7701", "push-manager base URL")
	flag.Parse()

	meta, err := hackcfg.Load(*cfgFlag, 7706)
	if err != nil {
		log.Printf("config: %v (using defaults)", err)
		meta.Port = 7706
	}

	cfgPath := configPath(*cfgFlag)
	app := newApp(cfgPath, loadConfig(cfgPath), *pmBase)

	// MIDI (idle detection + LED output) after boot-settle, then the one-time
	// startup LED sweep (only if enabled in config), then the idle state
	// machine and dependency watcher.
	go func() {
		app.setupMIDI()
		app.mu.Lock()
		sweep := app.cfg.StartupSweep
		app.mu.Unlock()
		if sweep {
			app.ledStartupSequence()
		}
		go app.watchIdle()
		go app.depWatch()
	}()

	http.HandleFunc("/", app.handleRoot)
	http.HandleFunc("/api/config", app.handleConfig)

	addr := fmt.Sprintf(":%d", meta.Port)
	log.Printf("screensaver listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// status is the JSON returned to the browser config page (/api/config).
type status struct {
	Config
	Active     bool     `json:"active"`
	Animations []string `json:"animations"`
	AnimName   string   `json:"animation_name"`
}

func (a *App) statusLocked() status {
	// caller holds a.mu
	name := ""
	if a.cfg.Animation >= 0 && a.cfg.Animation < len(animNames) {
		name = animNames[a.cfg.Animation]
	}
	return status{
		Config:     a.cfg,
		Active:     a.active,
		Animations: animNames,
		AnimName:   name,
	}
}

func (a *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}

// configPatch is a partial update — only present fields are applied, so the
// panel can change one setting at a time.
type configPatch struct {
	Enabled      *bool `json:"enabled"`
	Animation    *int  `json:"animation"`
	IdleSeconds  *int  `json:"idle_seconds"`
	Speed        *int  `json:"speed"`
	StartupSweep *bool `json:"startup_sweep"`
}

func (a *App) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var p configPatch
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		a.mu.Lock()
		if p.Enabled != nil {
			a.cfg.Enabled = *p.Enabled
		}
		if p.Animation != nil {
			a.cfg.Animation = *p.Animation
		}
		if p.IdleSeconds != nil {
			a.cfg.IdleSeconds = *p.IdleSeconds
		}
		if p.Speed != nil {
			a.cfg.Speed = *p.Speed
		}
		if p.StartupSweep != nil {
			a.cfg.StartupSweep = *p.StartupSweep
		}
		a.cfg.clamp()
		cfg := a.cfg
		a.mu.Unlock()
		a.persist(cfg)
	}

	a.mu.Lock()
	s := a.statusLocked()
	a.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s)
}

// persist writes cfg to disk off the hot-path lock.
func (a *App) persist(cfg Config) {
	saveMu.Lock()
	if err := saveConfig(a.cfgPath, cfg); err != nil {
		log.Printf("config save: %v", err)
	}
	saveMu.Unlock()
}
