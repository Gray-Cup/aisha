package state

import (
	"fmt"
	"net"
	"sync"
	"time"

	"twisha/internal/config"
)

// ─────────────────────────────────────────────
//  Health tracker
// ─────────────────────────────────────────────

// HealthInfo is a point-in-time snapshot of one project's health.
type HealthInfo struct {
	Up      bool
	Latency time.Duration
}

// HealthStatus tracks TCP reachability for every project.
type HealthStatus struct {
	mu      sync.RWMutex
	up      map[string]bool
	latency map[string]time.Duration
	checked map[string]time.Time
}

func NewStatus() *HealthStatus {
	return &HealthStatus{
		up:      make(map[string]bool),
		latency: make(map[string]time.Duration),
		checked: make(map[string]time.Time),
	}
}

// Probe performs a TCP dial to p.Port and records the result.
func (s *HealthStatus) Probe(p config.Project) {
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

// IsUp returns whether the named project was last seen reachable.
func (s *HealthStatus) IsUp(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.up[name]
}

// Snapshot returns a copy of health data for the given project names.
func (s *HealthStatus) Snapshot(names []string) map[string]HealthInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]HealthInfo, len(names))
	for _, name := range names {
		out[name] = HealthInfo{Up: s.up[name], Latency: s.latency[name]}
	}
	return out
}
