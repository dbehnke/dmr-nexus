package metrics

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

// PrometheusConfig holds Prometheus server configuration
type PrometheusConfig struct {
	Enabled bool
	Port    int
	Path    string
}

// PrometheusHandler handles Prometheus metrics HTTP requests
type PrometheusHandler struct {
	collector *Collector
}

// NewPrometheusHandler creates a new Prometheus handler
func NewPrometheusHandler(collector *Collector) *PrometheusHandler {
	return &PrometheusHandler{
		collector: collector,
	}
}

// ServeHTTP handles HTTP requests for metrics
func (h *PrometheusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	var output strings.Builder

	// Peer metrics
	output.WriteString("# HELP dmr_peers_total Total number of peer connections\n")
	output.WriteString("# TYPE dmr_peers_total counter\n")
	output.WriteString(fmt.Sprintf("dmr_peers_total %d\n", h.collector.GetTotalPeers()))

	output.WriteString("# HELP dmr_peers_active Number of currently active peers\n")
	output.WriteString("# TYPE dmr_peers_active gauge\n")
	output.WriteString(fmt.Sprintf("dmr_peers_active %d\n", h.collector.GetActivePeers()))

	// Packet metrics
	output.WriteString("# HELP dmr_packets_received_total Total packets received\n")
	output.WriteString("# TYPE dmr_packets_received_total counter\n")
	output.WriteString(fmt.Sprintf("dmr_packets_received_total %d\n", h.collector.GetPacketsReceived()))

	output.WriteString("# HELP dmr_packets_sent_total Total packets sent\n")
	output.WriteString("# TYPE dmr_packets_sent_total counter\n")
	output.WriteString(fmt.Sprintf("dmr_packets_sent_total %d\n", h.collector.GetPacketsSent()))

	// Byte metrics
	output.WriteString("# HELP dmr_bytes_received_total Total bytes received\n")
	output.WriteString("# TYPE dmr_bytes_received_total counter\n")
	output.WriteString(fmt.Sprintf("dmr_bytes_received_total %d\n", h.collector.GetBytesReceived()))

	output.WriteString("# HELP dmr_bytes_sent_total Total bytes sent\n")
	output.WriteString("# TYPE dmr_bytes_sent_total counter\n")
	output.WriteString(fmt.Sprintf("dmr_bytes_sent_total %d\n", h.collector.GetBytesSent()))

	// Stream metrics
	output.WriteString("# HELP dmr_streams_active Number of active voice streams\n")
	output.WriteString("# TYPE dmr_streams_active gauge\n")
	output.WriteString(fmt.Sprintf("dmr_streams_active %d\n", h.collector.GetActiveStreams()))

	// Bridge metrics
	output.WriteString("# HELP dmr_bridge_routes_total Total bridge routing events\n")
	output.WriteString("# TYPE dmr_bridge_routes_total counter\n")
	output.WriteString(fmt.Sprintf("dmr_bridge_routes_total %d\n", h.collector.GetBridgeRoutes()))

	// Talkgroup metrics
	output.WriteString("# HELP dmr_talkgroups_active Number of active talkgroups\n")
	output.WriteString("# TYPE dmr_talkgroups_active gauge\n")
	output.WriteString(fmt.Sprintf("dmr_talkgroups_active %d\n", h.collector.GetActiveTalkgroups()))

	w.Write([]byte(output.String()))
}

// PrometheusServer is an HTTP server for Prometheus metrics
type PrometheusServer struct {
	config    PrometheusConfig
	collector *Collector
	log       *logger.Logger
	server    *http.Server
}

// NewPrometheusServer creates a new Prometheus metrics server
func NewPrometheusServer(config PrometheusConfig, collector *Collector, log *logger.Logger) *PrometheusServer {
	if log == nil {
		log = logger.New(logger.Config{Level: "info", Format: "text"})
	}

	return &PrometheusServer{
		config:    config,
		collector: collector,
		log:       log.WithComponent("metrics"),
	}
}

// Start starts the Prometheus metrics server
func (s *PrometheusServer) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.log.Info("Prometheus metrics server disabled")
		return nil
	}

	handler := NewPrometheusHandler(s.collector)
	mux := http.NewServeMux()
	mux.Handle(s.config.Path, handler)

	// Use a listener to get the actual port (useful for testing with port 0)
	addr := fmt.Sprintf(":%d", s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port

	s.server = &http.Server{
		Handler: mux,
	}

	s.log.Info("Starting Prometheus metrics server",
		logger.Int("port", actualPort),
		logger.String("path", s.config.Path))

	// Start server
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		s.log.Info("Shutting down Prometheus metrics server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("metrics server shutdown error: %w", err)
		}
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// Stop stops the Prometheus metrics server
func (s *PrometheusServer) Stop() {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
	}
}
