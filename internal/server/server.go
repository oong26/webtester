package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"time"

	"webtester/internal/runner"
)

// Handler serves the web UI from static and exposes POST /api/run.
func Handler(static fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/run", handleRun)
	mux.Handle("/", http.FileServer(http.FS(static)))
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	// CSRF guard: this tool sends traffic to arbitrary targets, so only accept
	// JSON requests that come from this UI (same origin) and not from other websites.
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "Content-Type must be application/json"})
		return
	}
	if o := r.Header.Get("Origin"); o != "" {
		if u, err := url.Parse(o); err != nil || u.Host != r.Host {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "cross-origin request rejected"})
			return
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req runner.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	rep, err := runner.Run(ctx, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rep)
}
