package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"aisha/internal/config"
	"aisha/internal/network"
	"aisha/internal/proxy"
	"aisha/internal/state"
)

//go:embed all:frontend
var assets embed.FS

func main() {
	// ── Parse flags ──────────────────────────────────────────────────────
	headless := false
	var filtered []string
	for _, a := range os.Args[1:] {
		if a == "--headless" {
			headless = true
		} else {
			filtered = append(filtered, a)
		}
	}

	cfgPath := "config.json"
	if len(filtered) > 0 {
		cfgPath = filtered[0]
	}
	if !filepath.IsAbs(cfgPath) {
		exe, _ := os.Executable()
		cfgPath = filepath.Join(filepath.Dir(exe), cfgPath)
	}

	// ── Load (or seed) config ────────────────────────────────────────────
	cfg, err := config.Load(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			def := config.Default()
			data, _ := json.MarshalIndent(def, "", "  ")
			_ = os.WriteFile(cfgPath, data, 0644)
			log.Printf("Created default config at %s", cfgPath)
			cfg = def
		} else {
			log.Fatalf("Config error: %v", err)
		}
	}

	if cfg.LogFile != "" {
		if f, ferr := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); ferr == nil {
			log.SetOutput(f)
		}
	}

	// ── Build shared runtime state ───────────────────────────────────────
	ip := network.LocalIP()
	log.Printf("Aisha starting on %s (headless=%v)", ip, headless)

	stat := state.NewStatus()
	trk := state.NewTracker()
	mac := state.NewMACRules(cfg)
	arp := network.NewARPCache()
	pm := state.NewProcManager()

	// ── Start proxy server (network-wide, all interfaces) ───────────────
	ph := proxy.New(cfg, stat, trk, mac, arp, ip)
	proxySrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ProxyPort),
		Handler: ph,
	}
	go func() {
		if err := proxySrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Proxy error: %v", err)
		}
	}()
	log.Printf("Proxy listening on :%d  (routes *.local → localhost ports)", cfg.ProxyPort)

	// ── Health-probe loop ────────────────────────────────────────────────
	for _, p := range cfg.Projects {
		go stat.Probe(p)
	}
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			for _, p := range ph.GetConfig().Projects {
				go stat.Probe(p)
			}
		}
	}()

	// ── Headless daemon mode (no window) ─────────────────────────────────
	if headless {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		pm.StopAll()
		log.Println("Aisha shut down.")
		return
	}

	// ── Wails desktop window ─────────────────────────────────────────────
	app := NewApp(ph, cfgPath, stat, trk, mac, pm, ip)

	if err := wails.Run(&options.App{
		Title:            "Aisha — Local Network Proxy",
		Width:            1100,
		Height:           720,
		MinWidth:         800,
		MinHeight:        500,
		DisableResize:    false,
		Fullscreen:       false,
		BackgroundColour: &options.RGBA{R: 243, G: 243, B: 243, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: func(_ context.Context) { pm.StopAll() },
		Bind:       []interface{}{app},
	}); err != nil {
		log.Fatalf("Wails error: %v", err)
	}
}
