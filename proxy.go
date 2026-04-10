package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
)

// ─────────────────────────────────────────────
//  Reverse proxy handler
// ─────────────────────────────────────────────

type proxyHandler struct {
	mu       sync.RWMutex
	cfg      Config
	routes   map[string]*httputil.ReverseProxy
	projects map[string]Project
	stat     *healthStatus
	trk      *tracker
	mac      *macRules
	arp      *arpCache
	ip       string
}

func newProxyHandler(cfg Config, stat *healthStatus, trk *tracker, mac *macRules, arp *arpCache, ip string) *proxyHandler {
	h := &proxyHandler{
		cfg:      cfg,
		routes:   make(map[string]*httputil.ReverseProxy),
		projects: make(map[string]Project),
		stat:     stat,
		trk:      trk,
		mac:      mac,
		arp:      arp,
		ip:       ip,
	}
	for _, p := range cfg.Projects {
		h.buildRoute(p)
	}
	return h
}

// buildRoute creates/replaces the reverse proxy entry for p.
// Must be called with h.mu held (write) or during construction.
func (h *proxyHandler) buildRoute(p Project) {
	target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", p.Port))
	rp := httputil.NewSingleHostReverseProxy(target)
	pCopy := p
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w,
			fmt.Sprintf("%s is not reachable (port %d): %v", pCopy.Name, pCopy.Port, err),
			http.StatusBadGateway,
		)
	}
	key := p.Name + ".local"
	h.routes[key] = rp
	h.projects[key] = p
}

// addProject inserts a new project into the live routing table and config.
func (h *proxyHandler) addProject(p Project) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.buildRoute(p)
	h.cfg.Projects = append(h.cfg.Projects, p)
}

// updateProject replaces port/command/dir for an existing project in-place.
// Returns false if the project name is not found.
func (h *proxyHandler) updateProject(name string, port int, command, dir string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, p := range h.cfg.Projects {
		if p.Name == name {
			h.cfg.Projects[i].Port = port
			h.cfg.Projects[i].Command = command
			h.cfg.Projects[i].Dir = dir
			h.buildRoute(h.cfg.Projects[i])
			return true
		}
	}
	return false
}

// getConfig returns a snapshot of the current config.
func (h *proxyHandler) getConfig() Config {
	h.mu.RLock()
	defer h.mu.RUnlock()
	// Deep-copy projects slice so callers can't mutate shared state.
	cfg := h.cfg
	cfg.Projects = make([]Project, len(h.cfg.Projects))
	copy(cfg.Projects, h.cfg.Projects)
	return cfg
}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.ToLower(host)

	h.mu.RLock()
	proxy, ok := h.routes[host]
	proj := h.projects[host]
	routes := h.routes
	serverIP := h.ip
	adminPort := h.cfg.AdminPort
	h.mu.RUnlock()

	if !ok {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, notFoundPage(host, routes, serverIP))
		return
	}

	clientIP := extractClientIP(r)
	clientMAC := ""
	if clientIP != "127.0.0.1" && clientIP != "::1" && clientIP != serverIP {
		clientMAC = h.arp.lookup(clientIP)
	}

	if !h.mac.allowed(proj.Name, clientMAC) {
		h.trk.record(proj.Name, clientIP, clientMAC, r.URL.Path, http.StatusForbidden)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, forbiddenPage(proj.Name, clientIP, clientMAC, serverIP, adminPort))
		return
	}

	h.trk.record(proj.Name, clientIP, clientMAC, r.URL.Path, http.StatusOK)
	w.Header().Set("X-Proxied-By", "Twisha")
	proxy.ServeHTTP(w, r)
}
