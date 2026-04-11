package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aisha/internal/config"
	"aisha/internal/proxy"
	"aisha/internal/state"
)

// ─────────────────────────────────────────────
//  Wails App — all methods here are bound to
//  the frontend via window.go.main.App.*
// ─────────────────────────────────────────────

// App holds shared runtime state and exposes every admin action as a
// bound method callable from the Wails frontend.
type App struct {
	ctx     context.Context
	ph      *proxy.Handler
	cfgPath string
	stat    *state.HealthStatus
	trk     *state.Tracker
	mac     *state.MACRules
	pm      *state.ProcManager
	ip      string
}

func NewApp(
	ph *proxy.Handler,
	cfgPath string,
	stat *state.HealthStatus,
	trk *state.Tracker,
	mac *state.MACRules,
	pm *state.ProcManager,
	ip string,
) *App {
	return &App{ph: ph, cfgPath: cfgPath, stat: stat, trk: trk, mac: mac, pm: pm, ip: ip}
}

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// ── Return-type definitions ───────────────────────────────────────

type StatusRow struct {
	Name    string `json:"name"`
	Port    int    `json:"port"`
	Up      bool   `json:"up"`
	Domain  string `json:"domain"`
	Command string `json:"command,omitempty"`
	Dir     string `json:"dir,omitempty"`
	Managed bool   `json:"managed"`
	Latency int64  `json:"latency_ms"`
}

type RequestsResponse struct {
	Stats  []state.ProjStats `json:"stats"`
	Recent []state.ReqEntry  `json:"recent"`
}

type ServerInfo struct {
	IP        string `json:"ip"`
	ProxyPort int    `json:"proxy_port"`
}

type DirEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Up    bool   `json:"up"`
}

type BrowseResult struct {
	Path    string     `json:"path"`
	Entries []DirEntry `json:"entries"`
}

// ── Bound methods ─────────────────────────────────────────────────

// GetServerInfo returns the host IP and proxy port for the sidebar.
func (a *App) GetServerInfo() ServerInfo {
	cfg := a.ph.GetConfig()
	return ServerInfo{IP: a.ip, ProxyPort: cfg.ProxyPort}
}

// GetProjects returns the full project list from the live config.
func (a *App) GetProjects() []config.Project {
	return a.ph.GetConfig().Projects
}

// GetStatus returns health + managed state for every project.
func (a *App) GetStatus() []StatusRow {
	cfg := a.ph.GetConfig()
	names := make([]string, len(cfg.Projects))
	for i, p := range cfg.Projects {
		names[i] = p.Name
	}
	health := a.stat.Snapshot(names)
	rows := make([]StatusRow, 0, len(cfg.Projects))
	for _, p := range cfg.Projects {
		h := health[p.Name]
		rows = append(rows, StatusRow{
			Name:    p.Name,
			Port:    p.Port,
			Up:      h.Up,
			Domain:  fmt.Sprintf("http://%s.local", p.Name),
			Command: p.Command,
			Dir:     p.Dir,
			Managed: a.pm.IsRunning(p.Name),
			Latency: h.Latency.Milliseconds(),
		})
	}
	return rows
}

// GetRequests returns per-project stats and the recent access log.
func (a *App) GetRequests() RequestsResponse {
	cfg := a.ph.GetConfig()
	stats, recent := a.trk.Stats(cfg.Projects, a.mac.Get)
	if stats == nil {
		stats = []state.ProjStats{}
	}
	if recent == nil {
		recent = []state.ReqEntry{}
	}
	return RequestsResponse{Stats: stats, Recent: recent}
}

// CreateProject adds a new project to the live proxy and config file.
func (a *App) CreateProject(name string, port int, command, dir string) error {
	cfg := a.ph.GetConfig()
	for _, p := range cfg.Projects {
		if p.Name == name {
			return fmt.Errorf("project %q already exists", name)
		}
	}
	newProj := config.Project{Name: name, Port: port, Command: command, Dir: dir}
	a.ph.AddProject(newProj)
	_ = config.Save(a.cfgPath, a.ph.GetConfig())
	go a.stat.Probe(newProj)
	return nil
}

// UpdateProject replaces port/command/dir for an existing project.
func (a *App) UpdateProject(name string, port int, command, dir string) error {
	if !a.ph.UpdateProject(name, port, command, dir) {
		return fmt.Errorf("project %q not found", name)
	}
	_ = config.Save(a.cfgPath, a.ph.GetConfig())
	return nil
}

// DeleteProject removes a project from the proxy and config file.
func (a *App) DeleteProject(name string) error {
	if !a.ph.DeleteProject(name) {
		return fmt.Errorf("project %q not found", name)
	}
	_ = config.Save(a.cfgPath, a.ph.GetConfig())
	return nil
}

// StartProject launches a project's command via the process manager.
func (a *App) StartProject(name, command, dir string) error {
	if command != "" {
		a.ph.UpdateProject(name, 0, command, dir)
		_ = config.Save(a.cfgPath, a.ph.GetConfig())
	}
	cfg := a.ph.GetConfig()
	for _, p := range cfg.Projects {
		if p.Name == name {
			return a.pm.Start(p)
		}
	}
	return fmt.Errorf("project %q not found", name)
}

// StopProject sends SIGTERM to the project's managed process.
func (a *App) StopProject(name string) error {
	return a.pm.Stop(name)
}

// GetOutput returns the buffered stdout/stderr lines for a project.
func (a *App) GetOutput(name string) []string {
	lines := a.pm.GetOutput(name)
	if lines == nil {
		return []string{}
	}
	return lines
}

// SetMACRules updates the MAC allowlist for a project and persists it.
func (a *App) SetMACRules(project string, macs []string) error {
	a.mac.Set(project, macs)
	cfgCopy := a.ph.GetConfig()
	snap := a.mac.Snapshot()
	for i, p := range cfgCopy.Projects {
		cfgCopy.Projects[i].AllowedMACs = snap[p.Name]
	}
	return config.Save(a.cfgPath, cfgCopy)
}

// BrowseDir returns the subdirectory listing for the folder picker.
func (a *App) BrowseDir(path string) (BrowseResult, error) {
	if path == "" {
		if home, err := os.UserHomeDir(); err == nil {
			path = home
		} else {
			path = "/"
		}
	}
	path = filepath.Clean(path)
	entries, err := os.ReadDir(path)
	if err != nil {
		return BrowseResult{}, err
	}
	result := make([]DirEntry, 0)
	parent := filepath.Dir(path)
	if parent != path {
		result = append(result, DirEntry{Name: "..", Path: parent, IsDir: true, Up: true})
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") || !e.IsDir() {
			continue
		}
		result = append(result, DirEntry{
			Name:  e.Name(),
			Path:  filepath.Join(path, e.Name()),
			IsDir: true,
			Up:    false,
		})
	}
	return BrowseResult{Path: path, Entries: result}, nil
}
