package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ─────────────────────────────────────────────
//  Admin HTTP handler
// ─────────────────────────────────────────────

func adminHandler(proxy *proxyHandler, cfgPath string, stat *healthStatus, trk *tracker, mac *macRules, pm *procManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow webview same-origin fetches
		w.Header().Set("Access-Control-Allow-Origin", "*")

		cfg := proxy.getConfig()

		switch r.URL.Path {

		// ── GET /api/status ──────────────────────────────────────────────
		case "/api/status":
			w.Header().Set("Content-Type", "application/json")
			type row struct {
				Name    string `json:"name"`
				Port    int    `json:"port"`
				Up      bool   `json:"up"`
				Domain  string `json:"domain"`
				Command string `json:"command,omitempty"`
				Dir     string `json:"dir,omitempty"`
				Managed bool   `json:"managed"`
			}
			stat.mu.RLock()
			rows := make([]row, 0, len(cfg.Projects))
			for _, p := range cfg.Projects {
				rows = append(rows, row{
					Name:    p.Name,
					Port:    p.Port,
					Up:      stat.up[p.Name],
					Domain:  fmt.Sprintf("http://%s.local", p.Name),
					Command: p.Command,
					Dir:     p.Dir,
					Managed: pm.isRunning(p.Name),
				})
			}
			stat.mu.RUnlock()
			json.NewEncoder(w).Encode(rows)

		// ── GET /api/requests ────────────────────────────────────────────
		case "/api/requests":
			w.Header().Set("Content-Type", "application/json")
			type pStats struct {
				Name        string   `json:"name"`
				Total       int64    `json:"total"`
				Denied      int64    `json:"denied"`
				AllowedMACs []string `json:"allowed_macs"`
			}
			trk.mu.RLock()
			stats := make([]pStats, 0, len(cfg.Projects))
			for _, p := range cfg.Projects {
				stats = append(stats, pStats{
					Name:        p.Name,
					Total:       trk.counts[p.Name],
					Denied:      trk.denied[p.Name],
					AllowedMACs: mac.get(p.Name),
				})
			}
			recent := make([]ReqEntry, len(trk.recent))
			for i, e := range trk.recent {
				recent[len(trk.recent)-1-i] = e
			}
			trk.mu.RUnlock()
			type resp struct {
				Stats  []pStats   `json:"stats"`
				Recent []ReqEntry `json:"recent"`
			}
			json.NewEncoder(w).Encode(resp{Stats: stats, Recent: recent})

		// ── POST /api/mac-rules ──────────────────────────────────────────
		case "/api/mac-rules":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				Project string   `json:"project"`
				MACs    []string `json:"macs"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			mac.set(req.Project, req.MACs)
			cfgCopy := proxy.getConfig()
			snap := mac.snapshot()
			for i, p := range cfgCopy.Projects {
				cfgCopy.Projects[i].AllowedMACs = snap[p.Name]
			}
			_ = saveConfig(cfgPath, cfgCopy)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true}`)

		// ── POST /api/start ──────────────────────────────────────────────
		case "/api/start":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				Project string `json:"project"`
				Command string `json:"command"`
				Dir     string `json:"dir"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			// If the caller supplies updated command/dir, persist them first.
			if req.Command != "" {
				proxy.updateProject(req.Project, 0, req.Command, req.Dir)
				_ = saveConfig(cfgPath, proxy.getConfig())
				cfg = proxy.getConfig()
			}
			var found *Project
			for i := range cfg.Projects {
				if cfg.Projects[i].Name == req.Project {
					found = &cfg.Projects[i]
					break
				}
			}
			if found == nil {
				http.Error(w, "project not found", http.StatusNotFound)
				return
			}
			if err := pm.start(*found); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, `{"ok":false,"error":%q}`, err.Error())
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true}`)

		// ── POST /api/stop ───────────────────────────────────────────────
		case "/api/stop":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct{ Project string `json:"project"` }
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if err := pm.stop(req.Project); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, `{"ok":false,"error":%q}`, err.Error())
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true}`)

		// ── GET /api/output?project=… ────────────────────────────────────
		case "/api/output":
			w.Header().Set("Content-Type", "application/json")
			project := r.URL.Query().Get("project")
			lines := pm.getOutput(project)
			if lines == nil {
				lines = []string{}
			}
			json.NewEncoder(w).Encode(lines)

		// ── POST /api/projects  (create new project) ─────────────────────
		case "/api/projects":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				Name    string `json:"name"`
				Port    int    `json:"port"`
				Command string `json:"command"`
				Dir     string `json:"dir"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if req.Name == "" || req.Port == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, `{"ok":false,"error":"name and port are required"}`)
				return
			}
			// Guard against duplicate names
			for _, p := range cfg.Projects {
				if p.Name == req.Name {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, `{"ok":false,"error":"project %q already exists"}`, req.Name)
					return
				}
			}
			newProj := Project{
				Name:    req.Name,
				Port:    req.Port,
				Command: req.Command,
				Dir:     req.Dir,
			}
			proxy.addProject(newProj)
			_ = saveConfig(cfgPath, proxy.getConfig())
			go stat.probe(newProj)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true}`)

		// ── POST /api/update-project  (edit existing project) ────────────
		case "/api/update-project":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				Name    string `json:"name"`
				Port    int    `json:"port"`
				Command string `json:"command"`
				Dir     string `json:"dir"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if !proxy.updateProject(req.Name, req.Port, req.Command, req.Dir) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, `{"ok":false,"error":"project %q not found"}`, req.Name)
				return
			}
			_ = saveConfig(cfgPath, proxy.getConfig())
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true}`)

		// ── Dashboard (catch-all) ────────────────────────────────────────
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, dashboardPage(proxy.getConfig(), stat, proxy.ip))
		}
	})
}
