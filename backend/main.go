package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"aisha/backend/api"
	"aisha/backend/db"
	"aisha/backend/orchestrator"
	"aisha/backend/proxy"
)

func main() {
	dataDir := os.Getenv("TWISHA_DATA_DIR")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("cannot find home dir: %v", err)
		}
		dataDir = filepath.Join(home, ".aisha")
	}

	if err := os.MkdirAll(filepath.Join(dataDir, "logs"), 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	database, err := db.Init(filepath.Join(dataDir, "app.db"))
	if err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer database.Close()

	mgr := orchestrator.NewManager(database, dataDir)
	if err := mgr.RestoreState(); err != nil {
		log.Printf("warn: restore state: %v", err)
	}

	mux := http.NewServeMux()
	api.RegisterHandlers(mux, mgr, dataDir)

	// Serve the compiled React frontend for all non-API routes.
	// In dev mode the frontend dev server handles this; in production the
	// binary is embedded or served from the dist directory.
	staticDir := os.Getenv("TWISHA_STATIC_DIR")
	if staticDir != "" {
		fs := http.FileServer(http.Dir(staticDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// SPA fallback: serve index.html for unknown paths.
			if _, err := os.Stat(filepath.Join(staticDir, r.URL.Path)); os.IsNotExist(err) {
				http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
				return
			}
			fs.ServeHTTP(w, r)
		})
	}

	router := proxy.NewRouter(mgr)
	gateway := &proxy.Gateway{Router: router, APIMux: mux}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("shutting down…")
		mgr.StopAll()
		os.Exit(0)
	}()

	port := os.Getenv("AISHA_PORT")
	if port == "" {
		port = "3000"
	}
	addr := "0.0.0.0:" + port
	fmt.Printf("Aisha backend listening on %s\n", addr)
	fmt.Printf("Data dir: %s\n", dataDir)
	if err := http.ListenAndServe(addr, gateway); err != nil {
		log.Fatal(err)
	}
}
