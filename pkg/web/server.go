package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/config"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

// Server represents the web dashboard HTTP server
type Server struct {
	config config.WebConfig
	logger *logger.Logger
	server *http.Server
	hub    *WebSocketHub
	api    *API
	addr   string
	mu     sync.RWMutex
}

// NewServer creates a new web server instance
func NewServer(cfg config.WebConfig, log *logger.Logger) *Server {
	return &Server{
		config: cfg,
		logger: log,
		hub:    NewWebSocketHub(log),
		api:    NewAPI(log),
	}
}

// Start starts the web server
func Start(ctx context.Context, cfg config.WebConfig, log *logger.Logger) error {
	srv := NewServer(cfg, log)
	return srv.Start(ctx)
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.Info("Web server is disabled")
		return nil
	}

	// Start WebSocket hub
	go s.hub.Run(ctx)

	// Create HTTP router
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// API endpoints
	mux.HandleFunc("/api/status", s.api.HandleStatus)
	mux.HandleFunc("/api/peers", s.api.HandlePeers)
	mux.HandleFunc("/api/bridges", s.api.HandleBridges)
	mux.HandleFunc("/api/activity", s.api.HandleActivity)

	// WebSocket endpoint
	mux.Handle("/ws", s.hub.Handler())

	// Serve static frontend assets if present (frontend/dist)
	staticDir := "frontend/dist"
	// If the directory exists, mount a file handler with SPA fallback
	if fi, err := os.Stat(staticDir); err == nil && fi.IsDir() {
		s.logger.Info("Serving static frontend assets", logger.String("dir", staticDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Clean the path and try to serve the requested file
			reqPath := filepath.Clean(r.URL.Path)
			// Disallow path traversal outside staticDir
			if reqPath == "/" {
				http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
				return
			}
			// Trim leading '/'
			if len(reqPath) > 0 && reqPath[0] == '/' {
				reqPath = reqPath[1:]
			}
			fullPath := filepath.Join(staticDir, reqPath)
			if fi, err := os.Stat(fullPath); err == nil && !fi.IsDir() {
				http.ServeFile(w, r, fullPath)
				return
			}
			// Fallback to index.html for SPA routes
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
		})
	} else {
		s.logger.Info("No static frontend assets found; SPA not served", logger.String("dir", staticDir))
	}

	// Determine address
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Create HTTP server
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start listener to get actual address (especially for port 0)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	// Store the actual address
	s.mu.Lock()
	s.addr = listener.Addr().String()
	s.mu.Unlock()

	s.logger.Info("Starting web server",
		logger.String("address", s.addr))

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		s.logger.Info("Shutting down web server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shutdown server: %w", err)
		}
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// GetAddr returns the address the server is listening on
func (s *Server) GetAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.addr
}

// GetHub returns the WebSocket hub
func (s *Server) GetHub() *WebSocketHub {
	return s.hub
}

// handleHealth handles the health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "dmr-nexus",
		"time":    time.Now().Unix(),
	}); err != nil {
		s.logger.Warn("Failed to encode health response", logger.Error(err))
	}
}
