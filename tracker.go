package main

import (
	"net/http"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
//  Request tracker
// ─────────────────────────────────────────────

const maxRecent = 200

type ReqEntry struct {
	T       time.Time `json:"t"`
	Project string    `json:"project"`
	IP      string    `json:"ip"`
	MAC     string    `json:"mac"`
	Path    string    `json:"path"`
	Status  int       `json:"status"`
}

type tracker struct {
	mu     sync.RWMutex
	counts map[string]int64
	denied map[string]int64
	recent []ReqEntry
}

func newTracker() *tracker {
	return &tracker{
		counts: make(map[string]int64),
		denied: make(map[string]int64),
		recent: make([]ReqEntry, 0, maxRecent),
	}
}

func (t *tracker) record(project, ip, mac, path string, statusCode int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.counts[project]++
	if statusCode == http.StatusForbidden {
		t.denied[project]++
	}
	e := ReqEntry{T: time.Now(), Project: project, IP: ip, MAC: mac, Path: path, Status: statusCode}
	if len(t.recent) >= maxRecent {
		t.recent = t.recent[1:]
	}
	t.recent = append(t.recent, e)
}
