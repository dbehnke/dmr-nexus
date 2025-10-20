package metrics

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewPrometheusHandler tests creating a new handler
func TestNewPrometheusHandler(t *testing.T) {
	collector := NewCollector()
	handler := NewPrometheusHandler(collector)
	
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
}

// TestPrometheusHandler_ServeHTTP tests the HTTP handler
func TestPrometheusHandler_ServeHTTP(t *testing.T) {
	collector := NewCollector()
	handler := NewPrometheusHandler(collector)

	// Add some test data
	collector.PeerConnected(312000)
	collector.PacketReceived("DMRD")
	collector.BytesReceived(1024)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check that key metrics are present in output
	expectedMetrics := []string{
		"dmr_peers_total",
		"dmr_peers_active",
		"dmr_packets_received_total",
		"dmr_bytes_received_total",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(bodyStr, metric) {
			t.Errorf("Expected metric %s in output", metric)
		}
	}
}

// TestPrometheusHandler_Format tests metric format
func TestPrometheusHandler_Format(t *testing.T) {
	collector := NewCollector()
	handler := NewPrometheusHandler(collector)

	collector.PeerConnected(312000)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	bodyStr := string(body)

	// Check for basic Prometheus format (# HELP, # TYPE, metric lines)
	if !strings.Contains(bodyStr, "# HELP") {
		t.Error("Expected # HELP comments in output")
	}
	if !strings.Contains(bodyStr, "# TYPE") {
		t.Error("Expected # TYPE comments in output")
	}
}

// TestPrometheusServer tests starting and stopping the Prometheus server
func TestPrometheusServer(t *testing.T) {
	collector := NewCollector()
	config := PrometheusConfig{
		Enabled: true,
		Port:    0, // Use random port
		Path:    "/metrics",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := NewPrometheusServer(config, collector, nil)
	
	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	cancel()

	// Wait for server to stop
	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled && err != http.ErrServerClosed {
			t.Errorf("Unexpected error from server: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Server did not stop in time")
	}
}

// TestPrometheusServer_Disabled tests that disabled server doesn't start
func TestPrometheusServer_Disabled(t *testing.T) {
	collector := NewCollector()
	config := PrometheusConfig{
		Enabled: false,
	}

	ctx := context.Background()
	server := NewPrometheusServer(config, collector, nil)
	
	err := server.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error when disabled, got %v", err)
	}
}
