package network

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
//  IP helpers
// ─────────────────────────────────────────────

// LocalIP returns the primary non-loopback IPv4 address of this machine.
func LocalIP() string {
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

// ExtractClientIP pulls the real client IP from X-Forwarded-For or RemoteAddr.
func ExtractClientIP(r *http.Request) string {
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

// ARPCache maps IP addresses to MAC addresses with a 30-second TTL.
type ARPCache struct {
	mu      sync.RWMutex
	entries map[string]string
	updated time.Time
}

func NewARPCache() *ARPCache { return &ARPCache{entries: make(map[string]string)} }

// Lookup returns the MAC address for ip, refreshing the cache if stale.
func (a *ARPCache) Lookup(ip string) string {
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

func (a *ARPCache) refresh() {
	entries := make(map[string]string)
	// Linux: /proc/net/arp
	if data, err := os.ReadFile("/proc/net/arp"); err == nil {
		for _, line := range strings.Split(string(data), "\n")[1:] {
			fields := strings.Fields(line)
			if len(fields) >= 4 && fields[2] == "0x2" {
				entries[fields[0]] = strings.ToLower(fields[3])
			}
		}
	} else {
		// macOS / BSD: arp -an
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
