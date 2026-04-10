package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ─────────────────────────────────────────────
//  Config
// ─────────────────────────────────────────────

type Project struct {
	Name string `json:"name"` // e.g. "myapp"  → myapp.local
	Port int    `json:"port"` // e.g. 3000
}

type Config struct {
	Projects   []Project `json:"projects"`
	ProxyPort  int       `json:"proxy_port"`  // port the proxy listens on (default 80)
	AdminPort  int       `json:"admin_port"`  // status dashboard (default 9090)
	LogFile    string    `json:"log_file"`    // optional log path
}

func defaultConfig() Config {
	return Config{
		ProxyPort: 80,
		AdminPort: 9090,
		Projects: []Project{
			{Name: "myapp", Port: 3000},
			{Name: "api",   Port: 8080},
		},
	}
}

func loadConfig(path string) (Config, error) {
	cfg := defaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("invalid config JSON: %w", err)
	}
	if cfg.ProxyPort == 0 {
		cfg.ProxyPort = 80
	}
	if cfg.AdminPort == 0 {
		cfg.AdminPort = 9090
	}
	return cfg, nil
}

// ─────────────────────────────────────────────
//  Network helpers
// ─────────────────────────────────────────────

func localIP() string {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return "127.0.0.1"
}

// ─────────────────────────────────────────────
//  Health tracker
// ─────────────────────────────────────────────

type status struct {
	mu      sync.RWMutex
	up      map[string]bool
	latency map[string]time.Duration
	checked map[string]time.Time
}

func newStatus() *status {
	return &status{
		up:      make(map[string]bool),
		latency: make(map[string]time.Duration),
		checked: make(map[string]time.Time),
	}
}

func (s *status) probe(p Project) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", p.Port), 2*time.Second)
	lat := time.Since(start)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checked[p.Name] = time.Now()
	if err != nil {
		s.up[p.Name] = false
		return
	}
	conn.Close()
	s.up[p.Name] = true
	s.latency[p.Name] = lat
}

func (s *status) isUp(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.up[name]
}

// ─────────────────────────────────────────────
//  Reverse proxy handler
// ─────────────────────────────────────────────

type proxyHandler struct {
	routes map[string]*httputil.ReverseProxy // hostname → proxy
	names  map[string]int                    // hostname → port (for display)
	stat   *status
	ip     string
}

func newProxyHandler(cfg Config, stat *status, ip string) *proxyHandler {
	h := &proxyHandler{
		routes: make(map[string]*httputil.ReverseProxy),
		names:  make(map[string]int),
		stat:   stat,
		ip:     ip,
	}
	for _, p := range cfg.Projects {
		target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", p.Port))
		rp := httputil.NewSingleHostReverseProxy(target)
		rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, fmt.Sprintf("🔴 %s is not reachable (port %d): %v", p.Name, p.Port, err), http.StatusBadGateway)
		}
		// strip .local, handle host:port
		key := p.Name + ".local"
		h.routes[key] = rp
		h.names[key] = p.Port
	}
	return h
}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	// strip port suffix if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.ToLower(host)

	proxy, ok := h.routes[host]
	if !ok {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, notFoundPage(host, h.routes, h.ip))
		return
	}
	w.Header().Set("X-Proxied-By", "Twisha")
	proxy.ServeHTTP(w, r)
}

// ─────────────────────────────────────────────
//  Admin dashboard
// ─────────────────────────────────────────────

func adminHandler(cfg Config, stat *status, ip string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/status" {
			w.Header().Set("Content-Type", "application/json")
			type row struct {
				Name   string `json:"name"`
				Port   int    `json:"port"`
				Up     bool   `json:"up"`
				Domain string `json:"domain"`
			}
			var rows []row
			stat.mu.RLock()
			for _, p := range cfg.Projects {
				rows = append(rows, row{
					Name:   p.Name,
					Port:   p.Port,
					Up:     stat.up[p.Name],
					Domain: fmt.Sprintf("http://%s.local", p.Name),
				})
			}
			stat.mu.RUnlock()
			json.NewEncoder(w).Encode(rows)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, dashboardPage(cfg, stat, ip))
	})
}

// ─────────────────────────────────────────────
//  HTML pages
// ─────────────────────────────────────────────

func dashboardPage(cfg Config, stat *status, ip string) string {
	var rows strings.Builder
	stat.mu.RLock()
	defer stat.mu.RUnlock()
	for _, p := range cfg.Projects {
		up := stat.up[p.Name]
		badge := `<span class="badge up">● UP</span>`
		if !up {
			badge = `<span class="badge down">● DOWN</span>`
		}
		lat := stat.latency[p.Name]
		latStr := "—"
		if up {
			latStr = fmt.Sprintf("%dms", lat.Milliseconds())
		}
		rows.WriteString(fmt.Sprintf(`
		<tr>
			<td><a href="http://%s.local" target="_blank">%s.local</a></td>
			<td>:%d</td>
			<td>%s</td>
			<td>%s</td>
		</tr>`, p.Name, p.Name, p.Port, badge, latStr))
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Twisha Dashboard</title>
<meta http-equiv="refresh" content="5">
<style>
  *{box-sizing:border-box;margin:0;padding:0}
  body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#0f1117;color:#e2e8f0;padding:2rem}
  h1{font-size:1.8rem;margin-bottom:.25rem;color:#fff}
  .sub{color:#64748b;font-size:.9rem;margin-bottom:2rem}
  .card{background:#1e2130;border:1px solid #2d3148;border-radius:12px;overflow:hidden}
  table{width:100%%;border-collapse:collapse}
  th{background:#151724;padding:.75rem 1rem;text-align:left;font-size:.75rem;text-transform:uppercase;letter-spacing:.08em;color:#64748b}
  td{padding:.85rem 1rem;border-top:1px solid #2d3148;font-size:.9rem}
  a{color:#60a5fa;text-decoration:none}a:hover{text-decoration:underline}
  .badge{padding:.2rem .6rem;border-radius:999px;font-size:.75rem;font-weight:600}
  .up{background:#052e16;color:#4ade80}
  .down{background:#2d0a0a;color:#f87171}
  .info{margin-bottom:1.5rem;display:flex;gap:1rem;flex-wrap:wrap}
  .chip{background:#1e2130;border:1px solid #2d3148;border-radius:8px;padding:.5rem 1rem;font-size:.8rem}
  .chip span{color:#94a3b8}
</style>
</head>
<body>
<h1>🌊 Twisha</h1>
<p class="sub">Serving your localhost projects across the network — refreshes every 5s</p>
<div class="info">
  <div class="chip"><span>Mac IP: </span><strong>%s</strong></div>
  <div class="chip"><span>Proxy port: </span><strong>%d</strong></div>
  <div class="chip"><span>Projects: </span><strong>%d</strong></div>
</div>
<div class="card">
<table>
  <thead><tr><th>Domain</th><th>Local Port</th><th>Status</th><th>Latency</th></tr></thead>
  <tbody>%s</tbody>
</table>
</div>
</body></html>`, ip, cfg.ProxyPort, len(cfg.Projects), rows.String())
}

func notFoundPage(host string, routes map[string]*httputil.ReverseProxy, ip string) string {
	var links strings.Builder
	for k := range routes {
		links.WriteString(fmt.Sprintf(`<li><a href="http://%s">%s</a></li>`, k, k))
	}
	return fmt.Sprintf(`<!DOCTYPE html><html><head><title>Twisha – Not Found</title>
<style>body{font-family:-apple-system,sans-serif;background:#0f1117;color:#e2e8f0;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}
.box{max-width:480px;background:#1e2130;border:1px solid #2d3148;border-radius:12px;padding:2rem}
h1{font-size:1.4rem;color:#f87171;margin-bottom:.5rem}p{color:#94a3b8;margin:.5rem 0}ul{margin:.75rem 0 0 1.25rem}li{margin:.3rem 0}a{color:#60a5fa}</style>
</head><body><div class="box">
<h1>🔍 Unknown host: %s</h1>
<p>This domain isn't in your Twisha config.</p>
<p><strong>Available projects:</strong></p><ul>%s</ul>
<p style="margin-top:1rem;font-size:.8rem">Dashboard: <a href="http://%s:9090">%s:9090</a></p>
</div></body></html>`, host, links.String(), ip, ip)
}

// ─────────────────────────────────────────────
//  Entry point
// ─────────────────────────────────────────────

func main() {
	// Config path: first arg or default
	cfgPath := "config.json"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	// If not absolute, look next to the binary
	if !filepath.IsAbs(cfgPath) {
		exe, _ := os.Executable()
		cfgPath = filepath.Join(filepath.Dir(exe), cfgPath)
	}

	cfg, err := loadConfig(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Write a starter config
			def := defaultConfig()
			data, _ := json.MarshalIndent(def, "", "  ")
			_ = os.WriteFile(cfgPath, data, 0644)
			log.Printf("⚙️  Created default config at %s — edit it, then restart.", cfgPath)
			cfg = def
		} else {
			log.Fatalf("Config error: %v", err)
		}
	}

	// Setup logging
	if cfg.LogFile != "" {
		f, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			log.SetOutput(f)
		}
	}

	ip := localIP()
	log.Printf("🌊 Twisha starting on %s", ip)

	stat := newStatus()

	// Initial health probe
	for _, p := range cfg.Projects {
		go stat.probe(p)
	}

	// Periodic health checks every 10s
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			for _, p := range cfg.Projects {
				go stat.probe(p)
			}
		}
	}()

	// Reverse proxy server
	proxy := newProxyHandler(cfg, stat, ip)
	proxySrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ProxyPort),
		Handler: proxy,
	}

	// Admin dashboard
	adminSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.AdminPort),
		Handler: adminHandler(cfg, stat, ip),
	}

	// Print summary
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("  Proxy  → :%d", cfg.ProxyPort)
	log.Printf("  Dashboard → http://%s:%d", ip, cfg.AdminPort)
	log.Printf("  Projects:")
	for _, p := range cfg.Projects {
		log.Printf("    http://%s.local  →  localhost:%d", p.Name, p.Port)
	}
	log.Printf("  Your Mac IP: %s", ip)
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	go func() {
		if err := proxySrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Proxy error: %v", err)
		}
	}()
	go func() {
		if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Admin error: %v", err)
		}
	}()

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("Shutting down Twisha...")
}
