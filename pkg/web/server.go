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

	"github.com/dbehnke/dmr-nexus/pkg/bridge"
	"github.com/dbehnke/dmr-nexus/pkg/config"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/dbehnke/dmr-nexus/pkg/peer"
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

	// Optional dependencies for API data exposure
	peersProvider  interface{ GetAllPeers() []*peer.Peer }
	routerProvider interface {
		GetActiveBridges() []*bridge.BridgeRuleSet
	}
}

// spaHandler wraps an http.FileSystem to serve a Single Page Application.
// It tries to serve the requested file, and if not found, serves index.html instead.
// This is necessary for client-side routing (e.g., Vue Router with HTML5 history mode).
func spaHandler(fsys http.FileSystem) http.Handler {
	fileServer := http.FileServer(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to open the requested file
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		f, err := fsys.Open(path)
		if err == nil {
			// File exists, serve it normally
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// File not found, serve index.html for SPA routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
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

// WithPeerManager injects a PeerManager for API exposure
func (s *Server) WithPeerManager(pm *peer.PeerManager) *Server {
	s.peersProvider = pm
	if s.api != nil {
		s.api.SetDeps(pm, nil)
	}
	return s
}

// WithRouter injects a bridge Router for API exposure
func (s *Server) WithRouter(r *bridge.Router) *Server {
	s.routerProvider = r
	if s.api != nil {
		s.api.SetDeps(nil, r)
	}
	return s
}

// Start starts the web server
func Start(ctx context.Context, cfg config.WebConfig, log *logger.Logger) error {
	srv := NewServer(cfg, log)
	return srv.Start(ctx)
}

// StartWithDeps starts the web server with optional dependencies for API exposure
func StartWithDeps(ctx context.Context, cfg config.WebConfig, log *logger.Logger, pm *peer.PeerManager, r *bridge.Router) error {
	srv := NewServer(cfg, log)
	if pm != nil {
		srv.WithPeerManager(pm)
	}
	if r != nil {
		srv.WithRouter(r)
	}
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
	// Broadcast a lightweight heartbeat periodically so the UI can test realtime plumbing
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				s.hub.Broadcast(Event{
					Type:      "heartbeat",
					Timestamp: t,
					Data: map[string]interface{}{
						"clients": s.hub.GetClientCount(),
					},
				})
			}
		}
	}()

	// Wire API deps if provided
	if s.peersProvider != nil || s.routerProvider != nil {
		var pm *peer.PeerManager
		if p, ok := s.peersProvider.(*peer.PeerManager); ok {
			pm = p
		}
		var rt *bridge.Router
		if r, ok := s.routerProvider.(*bridge.Router); ok {
			rt = r
		}
		s.api.SetDeps(pm, rt)
	}

	// Create HTTP router
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// API endpoints
	mux.HandleFunc("/api/status", s.api.HandleStatus)
	mux.HandleFunc("/api/peers", s.api.HandlePeers)
	mux.HandleFunc("/api/bridges", s.api.HandleBridges)
	mux.HandleFunc("/api/activity", s.api.HandleActivity)
	mux.HandleFunc("/api/transmissions", s.api.HandleTransmissions)
	mux.HandleFunc("/api/user/", s.api.HandleUserLookup)

	// WebSocket endpoint
	mux.Handle("/ws", s.hub.Handler())

	// Try embedded static assets first (built into the binary via go:embed)
	if fsys, err := embeddedStaticFS(); err == nil && fsys != nil {
		s.logger.Info("Serving embedded frontend assets")
		mux.Handle("/", spaHandler(fsys))
	} else {
		// Fallback to filesystem directory
		staticDir := "frontend/dist"
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

// GetAPI returns the API instance
func (s *Server) GetAPI() *API {
	return s.api
}

// PeerConnectedHandler returns a function suitable for network server hook
func (s *Server) PeerConnectedHandler() func(id uint32, callsign string, addr string) {
	return func(id uint32, callsign string, addr string) {
		s.hub.BroadcastPeerConnected(id, callsign, addr)
	}
}

// PeerDisconnectedHandler returns a function suitable for network server hook
func (s *Server) PeerDisconnectedHandler() func(id uint32) {
	return func(id uint32) {
		s.hub.BroadcastPeerDisconnected(id)
	}
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
