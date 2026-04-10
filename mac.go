package main

import (
	"strings"
	"sync"
)

// ─────────────────────────────────────────────
//  Live MAC rules
// ─────────────────────────────────────────────

type macRules struct {
	mu    sync.RWMutex
	rules map[string][]string
}

func newMACRules(cfg Config) *macRules {
	m := &macRules{rules: make(map[string][]string)}
	for _, p := range cfg.Projects {
		if len(p.AllowedMACs) > 0 {
			m.rules[p.Name] = p.AllowedMACs
		}
	}
	return m
}

func (m *macRules) allowed(project, mac string) bool {
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

func (m *macRules) set(project string, macs []string) {
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

func (m *macRules) get(project string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp := make([]string, len(m.rules[project]))
	copy(cp, m.rules[project])
	return cp
}

func (m *macRules) snapshot() map[string][]string {
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
