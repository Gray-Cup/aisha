package state

import (
	"strings"
	"sync"

	"aisha/internal/config"
)

// ─────────────────────────────────────────────
//  MAC allowlist rules
// ─────────────────────────────────────────────

// MACRules holds per-project device allowlists, safe for concurrent use.
type MACRules struct {
	mu    sync.RWMutex
	rules map[string][]string
}

func NewMACRules(cfg config.Config) *MACRules {
	m := &MACRules{rules: make(map[string][]string)}
	for _, p := range cfg.Projects {
		if len(p.AllowedMACs) > 0 {
			m.rules[p.Name] = p.AllowedMACs
		}
	}
	return m
}

// Allowed returns true when no MAC filter is set for the project, or mac is listed.
func (m *MACRules) Allowed(project, mac string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rules, ok := m.rules[project]
	if !ok || len(rules) == 0 {
		return true
	}
	mac = strings.ToLower(strings.TrimSpace(mac))
	for _, r := range rules {
		if strings.ToLower(strings.TrimSpace(r)) == mac {
			return true
		}
	}
	return false
}

// Set replaces the allowlist for project. An empty slice removes the filter.
func (m *MACRules) Set(project string, macs []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var cleaned []string
	for _, mac := range macs {
		if mac = strings.ToLower(strings.TrimSpace(mac)); mac != "" {
			cleaned = append(cleaned, mac)
		}
	}
	if len(cleaned) == 0 {
		delete(m.rules, project)
	} else {
		m.rules[project] = cleaned
	}
}

// Get returns a copy of the allowlist for project.
func (m *MACRules) Get(project string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp := make([]string, len(m.rules[project]))
	copy(cp, m.rules[project])
	return cp
}

// Snapshot returns a deep copy of all rules.
func (m *MACRules) Snapshot() map[string][]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string][]string, len(m.rules))
	for k, v := range m.rules {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}
