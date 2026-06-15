// Minimal service used to demonstrate containerization. version is injected at
// build time via -ldflags "-X main.version=...".
package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
)

var version = "dev"

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"version": version})
	})

	addr := ":8080"
	log.Info("listening", "addr", addr, "version", version)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error("server error", "err", err)
		os.Exit(1)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
