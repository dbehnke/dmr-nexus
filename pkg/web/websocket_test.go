package web

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

func TestWebSocketHub_New(t *testing.T) {
	log := logger.New(logger.Config{Level: "info"})
	hub := NewWebSocketHub(log)

	if hub == nil {
		t.Fatal("NewWebSocketHub returned nil")
	}
}

func TestWebSocketHub_Run(t *testing.T) {
	log := logger.New(logger.Config{Level: "info"})
	hub := NewWebSocketHub(log)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start hub in goroutine
	go hub.Run(ctx)

	// Wait for hub to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context to stop hub
	cancel()

	// Wait a bit for hub to stop
	time.Sleep(50 * time.Millisecond)
}

func TestWebSocketHub_Broadcast(t *testing.T) {
	log := logger.New(logger.Config{Level: "info"})
	hub := NewWebSocketHub(log)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start hub
	go hub.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	// Create test event
	event := Event{
		Type: "test",
		Data: map[string]interface{}{"message": "hello"},
	}

	// Broadcast should not panic even with no clients
	hub.Broadcast(event)

	// Give time for broadcast to process
	time.Sleep(50 * time.Millisecond)
}

func TestWebSocketHandler(t *testing.T) {
	log := logger.New(logger.Config{Level: "info"})
	hub := NewWebSocketHub(log)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start hub
	go hub.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	// Create test server
	handler := hub.Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	// Get WebSocket URL
	_ = "ws" + strings.TrimPrefix(server.URL, "http")

	// Test connection (basic validation that handler is set up correctly)
	// Note: Full WebSocket test would require gorilla/websocket test client
	// For now, we validate handler setup
	if handler == nil {
		t.Fatal("WebSocket handler is nil")
	}
}

func TestEvent_Marshal(t *testing.T) {
	event := Event{
		Type:      "peer_connected",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"peer_id":  312000,
			"callsign": "W1ABC",
		},
	}

	data, err := event.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	if len(data) == 0 {
		t.Error("Marshaled data is empty")
	}

	// Should contain the type
	if !strings.Contains(string(data), "peer_connected") {
		t.Error("Marshaled data doesn't contain event type")
	}
}
