// Package health provides HTTP health check endpoints for monitoring XRP readiness.
//
// The health server runs on a separate port from the main proxy and provides:
// - GET /health endpoint that returns 102 Processing during startup
// - Returns 200 OK with body "ok" when the proxy is fully ready
//
// This enables external monitoring systems to determine when XRP is ready
// to handle traffic, particularly useful for container orchestration and
// load balancers that need to wait for plugin loading to complete.
package health

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

// Server provides health check endpoints for XRP
type Server struct {
	server *http.Server
	ready  *int32 // atomic flag for readiness state
}

// New creates a new health server on the specified port
func New(port int) *Server {
	var ready int32 // 0 = not ready, 1 = ready

	mux := http.NewServeMux()
	s := &Server{
		server: &http.Server{
			Addr:    ":" + strconv.Itoa(port),
			Handler: mux,
		},
		ready: &ready,
	}

	mux.HandleFunc("/health", s.healthHandler)

	return s
}

// Start begins listening for health check requests
func (s *Server) Start() error {
	slog.Info("Starting health server", "addr", s.server.Addr)
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the health server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// MarkReady sets the server state to ready, causing /health to return 200
func (s *Server) MarkReady() {
	atomic.StoreInt32(s.ready, 1)
	slog.Info("Health server marked as ready")
}

// MarkNotReady sets the server state to not ready, causing /health to return 102
func (s *Server) MarkNotReady() {
	atomic.StoreInt32(s.ready, 0)
	slog.Info("Health server marked as not ready")
}

// healthHandler handles GET /health requests
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if atomic.LoadInt32(s.ready) == 1 {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		if err != nil {
			slog.Error("Failed to write health response", "error", err)
		}
	} else {
		w.WriteHeader(http.StatusProcessing) // 102 Processing
		_, err := w.Write([]byte("starting"))
		if err != nil {
			slog.Error("Failed to write health response", "error", err)
		}
	}
}