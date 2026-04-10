package main

import (
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
//  Network helpers
// ─────────────────────────────────────────────

func localIP() string {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return "127.0.0.1"
}

func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ─────────────────────────────────────────────
//  ARP cache  (IP → MAC, refreshed every 30s)
// ─────────────────────────────────────────────

var reARP = regexp.MustCompile(`\((\d+\.\d+\.\d+\.\d+)\)\s+at\s+([0-9a-fA-F:]{17})`)

type arpCache struct {
	mu      sync.RWMutex
	entries map[string]string
	updated time.Time
}

func newARPCache() *arpCache { return &arpCache{entries: make(map[string]string)} }

func (a *arpCache) lookup(ip string) string {
	a.mu.RLock()
	fresh := time.Since(a.updated) < 30*time.Second
	mac := a.entries[ip]
	a.mu.RUnlock()
	if fresh {
		return mac
	}
	a.refresh()
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.entries[ip]
}

func (a *arpCache) refresh() {
	entries := make(map[string]string)
	if data, err := os.ReadFile("/proc/net/arp"); err == nil {
		for _, line := range strings.Split(string(data), "\n")[1:] {
			fields := strings.Fields(line)
			if len(fields) >= 4 && fields[2] == "0x2" {
				entries[fields[0]] = strings.ToLower(fields[3])
			}
		}
	} else {
		if out, err := exec.Command("arp", "-an").Output(); err == nil {
			for _, m := range reARP.FindAllSubmatch(out, -1) {
				entries[string(m[1])] = strings.ToLower(string(m[2]))
			}
		}
	}
	a.mu.Lock()
	a.entries = entries
	a.updated = time.Now()
	a.mu.Unlock()
}
