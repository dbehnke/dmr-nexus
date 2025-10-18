package web

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/config"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

func TestServer_New(t *testing.T) {
	cfg := config.WebConfig{
		Enabled:      true,
		Host:         "localhost",
		Port:         8080,
		AuthRequired: false,
	}

	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, log)

	if srv == nil {
		t.Fatal("NewServer returned nil")
	}

	if srv.config.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", srv.config.Port)
	}
}

func TestServer_StartStop(t *testing.T) {
	cfg := config.WebConfig{
		Enabled:      true,
		Host:         "localhost",
		Port:         0, // Use any available port
		AuthRequired: false,
	}

	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, log)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start(ctx)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop server
	cancel()

	// Wait for server to stop
	err := <-errChan
	if err != nil && err != context.Canceled && err != http.ErrServerClosed {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestServer_HealthEndpoint(t *testing.T) {
	cfg := config.WebConfig{
		Enabled:      true,
		Host:         "localhost",
		Port:         0, // Use any available port
		AuthRequired: false,
	}

	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server
	go func() {
		if err := srv.Start(ctx); err != nil && err != context.Canceled && err != http.ErrServerClosed {
			t.Logf("srv.Start error: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)

	// Get the actual address the server is listening on
	addr := srv.GetAddr()
	if addr == "" {
		t.Fatal("Server address is empty")
	}

	// Test health endpoint
	resp, err := http.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatalf("Failed to request health endpoint: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("resp.Body.Close error: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
