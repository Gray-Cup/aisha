package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// Gateway is the single HTTP handler that routes all traffic.
// API requests → APIMux.
// Subdomain / path requests → reverse proxy to the matching project.
// Everything else → APIMux (serves the dashboard SPA).
type Gateway struct {
	Router *Router
	APIMux http.Handler
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	addCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/api/") {
		g.APIMux.ServeHTTP(w, r)
		return
	}

	projectID, prefix, found := g.Router.Resolve(r)
	if !found {
		g.APIMux.ServeHTTP(w, r)
		return
	}

	port, err := g.Router.mgr.GetPort(projectID)
	if err != nil {
		log.Printf("proxy: project %q not found (path=%s)", projectID, r.URL.Path)
		http.Error(w, fmt.Sprintf("project %q not found", projectID), http.StatusNotFound)
		return
	}

	target := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("127.0.0.1:%d", port),
	}
	log.Printf("proxy: %s → 127.0.0.1:%d%s", projectID, port, r.URL.Path)

	if prefix != "" {
		r = r.Clone(r.Context())
		r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		if r.URL.RawPath != "" {
			r.URL.RawPath = strings.TrimPrefix(r.URL.RawPath, prefix)
		}
	}

	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		proxyWebSocket(w, r, target.Host)
		return
	}

	rp := httputil.NewSingleHostReverseProxy(target)
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
	}
	rp.ServeHTTP(w, r)
}

func addCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}
