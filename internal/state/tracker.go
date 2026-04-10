package state

import (
	"net/http"
	"sync"
	"time"

	"twisha/internal/config"
)

// ─────────────────────────────────────────────
//  Request tracker
// ─────────────────────────────────────────────

const MaxRecent = 200

// ReqEntry is one recorded proxy request.
type ReqEntry struct {
	T       time.Time `json:"t"`
	Project string    `json:"project"`
	IP      string    `json:"ip"`
	MAC     string    `json:"mac"`
	Path    string    `json:"path"`
	Status  int       `json:"status"`
}

// ProjStats aggregates counts for one project.
type ProjStats struct {
	Name        string   `json:"name"`
	Total       int64    `json:"total"`
	Denied      int64    `json:"denied"`
	AllowedMACs []string `json:"allowed_macs"`
}

// Tracker counts requests and keeps a recent ring-buffer.
type Tracker struct {
	mu     sync.RWMutex
	counts map[string]int64
	denied map[string]int64
	recent []ReqEntry
}

func NewTracker() *Tracker {
	return &Tracker{
		counts: make(map[string]int64),
		denied: make(map[string]int64),
		recent: make([]ReqEntry, 0, MaxRecent),
	}
}

// Record adds a request to the tracker.
func (t *Tracker) Record(project, ip, mac, path string, statusCode int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.counts[project]++
	if statusCode == http.StatusForbidden {
		t.denied[project]++
	}
	e := ReqEntry{T: time.Now(), Project: project, IP: ip, MAC: mac, Path: path, Status: statusCode}
	if len(t.recent) >= MaxRecent {
		t.recent = t.recent[1:]
	}
	t.recent = append(t.recent, e)
}

// Stats returns per-project aggregates and the recent log (newest first).
// getMacs is called to attach the current MAC allowlist to each project.
func (t *Tracker) Stats(projects []config.Project, getMacs func(string) []string) ([]ProjStats, []ReqEntry) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	stats := make([]ProjStats, 0, len(projects))
	for _, p := range projects {
		stats = append(stats, ProjStats{
			Name:        p.Name,
			Total:       t.counts[p.Name],
			Denied:      t.denied[p.Name],
			AllowedMACs: getMacs(p.Name),
		})
	}
	// Return recent entries newest-first.
	recent := make([]ReqEntry, len(t.recent))
	for i, e := range t.recent {
		recent[len(t.recent)-1-i] = e
	}
	return stats, recent
}
