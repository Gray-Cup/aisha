package proxy

import (
	"io"
	"net"
	"net/http"
)

// proxyWebSocket tunnels a WebSocket connection by hijacking both ends.
func proxyWebSocket(w http.ResponseWriter, r *http.Request, upstreamHost string) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket proxy unsupported", http.StatusInternalServerError)
		return
	}

	upstream, err := net.Dial("tcp", upstreamHost)
	if err != nil {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	defer upstream.Close()

	client, _, err := hj.Hijack()
	if err != nil {
		return
	}
	defer client.Close()

	// Forward the original HTTP upgrade request verbatim.
	r.Write(upstream)

	done := make(chan struct{}, 2)
	cp := func(dst, src net.Conn) {
		io.Copy(dst, src)
		done <- struct{}{}
	}
	go cp(upstream, client)
	go cp(client, upstream)
	<-done
}
