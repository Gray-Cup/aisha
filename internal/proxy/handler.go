package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"twisha/internal/config"
	"twisha/internal/network"
	"twisha/internal/state"
	"twisha/internal/ui"
)

// ─────────────────────────────────────────────
//  Reverse-proxy handler
// ─────────────────────────────────────────────

// Handler is the main HTTP handler that routes *.local requests to backend
// projects. Routes and config are protected by a RWMutex so projects can be
// added or updated without restarting.
type Handler struct {
	mu       sync.RWMutex
	cfg      config.Config
	routes   map[string]*httputil.ReverseProxy
	projects map[string]config.Project
	stat     *state.HealthStatus
	trk      *state.Tracker
	mac      *state.MACRules
	arp      *network.ARPCache
	ip       string
}

func New(
	cfg config.Config,
	stat *state.HealthStatus,
	trk *state.Tracker,
	mac *state.MACRules,
	arp *network.ARPCache,
	ip string,
) *Handler {
	h := &Handler{
		cfg:      cfg,
		routes:   make(map[string]*httputil.ReverseProxy),
		projects: make(map[string]config.Project),
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

// buildRoute creates or replaces the reverse-proxy entry for p.
// Must be called with h.mu held for writing, or during construction.
func (h *Handler) buildRoute(p config.Project) {
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

// AddProject inserts a new project into the live routing table.
func (h *Handler) AddProject(p config.Project) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.buildRoute(p)
	h.cfg.Projects = append(h.cfg.Projects, p)
}

// UpdateProject replaces port/command/dir for an existing project in-place.
// Returns false when the project name is not found.
func (h *Handler) UpdateProject(name string, port int, command, dir string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, p := range h.cfg.Projects {
		if p.Name == name {
			if port > 0 {
				h.cfg.Projects[i].Port = port
			}
			h.cfg.Projects[i].Command = command
			h.cfg.Projects[i].Dir = dir
			h.buildRoute(h.cfg.Projects[i])
			return true
		}
	}
	return false
}

// GetConfig returns a deep copy of the current config.
func (h *Handler) GetConfig() config.Config {
	h.mu.RLock()
	defer h.mu.RUnlock()
	cfg := h.cfg
	cfg.Projects = make([]config.Project, len(h.cfg.Projects))
	copy(cfg.Projects, h.cfg.Projects)
	return cfg
}

// IP returns the server's local IP address.
func (h *Handler) IP() string { return h.ip }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.ToLower(host)

	h.mu.RLock()
	rp, ok := h.routes[host]
	proj := h.projects[host]
	// Collect domain names for the 404 page while holding the read lock.
	domains := make([]string, 0, len(h.routes))
	for k := range h.routes {
		domains = append(domains, k)
	}
	serverIP := h.ip
	adminPort := h.cfg.AdminPort
	h.mu.RUnlock()

	if !ok {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, ui.NotFoundPage(host, domains, serverIP))
		return
	}

	clientIP := network.ExtractClientIP(r)
	clientMAC := ""
	if clientIP != "127.0.0.1" && clientIP != "::1" && clientIP != serverIP {
		clientMAC = h.arp.Lookup(clientIP)
	}

	if !h.mac.Allowed(proj.Name, clientMAC) {
		h.trk.Record(proj.Name, clientIP, clientMAC, r.URL.Path, http.StatusForbidden)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, ui.ForbiddenPage(proj.Name, clientIP, clientMAC, serverIP, adminPort))
		return
	}

	h.trk.Record(proj.Name, clientIP, clientMAC, r.URL.Path, http.StatusOK)
	w.Header().Set("X-Proxied-By", "Twisha")
	rp.ServeHTTP(w, r)
}
