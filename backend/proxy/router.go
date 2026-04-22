package proxy

import (
	"net/http"
	"strings"
)

// PortLookup is satisfied by orchestrator.Manager.
type PortLookup interface {
	GetPort(id string) (int, error)
}

type Router struct {
	mgr PortLookup
}

func NewRouter(mgr PortLookup) *Router {
	return &Router{mgr: mgr}
}

// Resolve returns (projectID, pathPrefix, found).
// pathPrefix is non-empty only for path-based routing — strip it before proxying.
func (r *Router) Resolve(req *http.Request) (projectID, pathPrefix string, found bool) {
	host := req.Host
	if i := strings.LastIndex(host, ":"); i != -1 {
		host = host[:i]
	}

	// Host-based: project1.localhost
	if strings.HasSuffix(host, ".localhost") {
		sub := strings.TrimSuffix(host, ".localhost")
		if sub != "" && sub != "localhost" {
			return sub, "", true
		}
	}

	// Path-based: /projectid/...  (never matches /api/...)
	path := strings.TrimPrefix(req.URL.Path, "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) > 0 && parts[0] != "" && parts[0] != "api" {
		return parts[0], "/" + parts[0], true
	}

	return "", "", false
}
