package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	webview "github.com/webview/webview_go"
)

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
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			def := defaultConfig()
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

	// ── Build shared state ───────────────────────────────────────────────
	ip := localIP()
	log.Printf("Twisha starting on %s (headless=%v)", ip, headless)

	stat := newStatus()
	trk := newTracker()
	mac := newMACRules(cfg)
	arp := newARPCache()
	pm := newProcManager()

	// ── Proxy + admin servers ────────────────────────────────────────────
	proxy := newProxyHandler(cfg, stat, trk, mac, arp, ip)

	proxySrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ProxyPort),
		Handler: proxy,
	}
	adminSrv := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", cfg.AdminPort),
		Handler: adminHandler(proxy, cfgPath, stat, trk, mac, pm),
	}

	go func() {
		if err := proxySrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Proxy error: %v", err)
		}
	}()
	go func() {
		if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Admin error: %v", err)
		}
	}()

	log.Printf("Proxy :%d  |  Dashboard http://127.0.0.1:%d", cfg.ProxyPort, cfg.AdminPort)

	// ── Health-probe loop (includes dynamically added projects) ──────────
	for _, p := range cfg.Projects {
		go stat.probe(p)
	}
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			for _, p := range proxy.getConfig().Projects {
				go stat.probe(p)
			}
		}
	}()

	// ── Headless daemon mode ─────────────────────────────────────────────
	if headless {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		pm.stopAll()
		log.Println("Twisha shut down.")
		return
	}

	// ── Wait for admin server to be ready, then open native window ───────
	adminAddr := fmt.Sprintf("127.0.0.1:%d", cfg.AdminPort)
	for i := 0; i < 40; i++ {
		conn, dialErr := net.DialTimeout("tcp", adminAddr, 100*time.Millisecond)
		if dialErr == nil {
			conn.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("Twisha — Local Network Proxy")
	w.SetSize(1100, 720, webview.HintNone)
	w.Navigate(fmt.Sprintf("http://127.0.0.1:%d", cfg.AdminPort))

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		pm.stopAll()
		w.Terminate()
	}()

	w.Run() // blocks until the window is closed
	pm.stopAll()
	log.Println("Twisha shut down.")
}
