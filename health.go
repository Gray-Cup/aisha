package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
//  Health tracker
// ─────────────────────────────────────────────

type healthStatus struct {
	mu      sync.RWMutex
	up      map[string]bool
	latency map[string]time.Duration
	checked map[string]time.Time
}

func newStatus() *healthStatus {
	return &healthStatus{
		up:      make(map[string]bool),
		latency: make(map[string]time.Duration),
		checked: make(map[string]time.Time),
	}
}

func (s *healthStatus) probe(p Project) {
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

func (s *healthStatus) isUp(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.up[name]
}
