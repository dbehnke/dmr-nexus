package web

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

// Event represents a WebSocket event to be broadcast to clients
type Event struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Marshal converts an event to JSON bytes
func (e *Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	messages chan []byte
	hub      *WebSocketHub
}

// WebSocketHub manages WebSocket client connections and broadcasts
type WebSocketHub struct {
	clients    map[*Client]bool
	broadcast  chan Event
	register   chan *Client
	unregister chan *Client
	logger     *logger.Logger
	mu         sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(log *logger.Logger) *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     log,
	}
}

// Run starts the WebSocket hub event loop
func (h *WebSocketHub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Debug("WebSocket client registered",
				logger.String("client_id", client.ID))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.messages)
			}
			h.mu.Unlock()
			h.logger.Debug("WebSocket client unregistered",
				logger.String("client_id", client.ID))

		case event := <-h.broadcast:
			// Marshal event to JSON
			data, err := event.Marshal()
			if err != nil {
				h.logger.Error("Failed to marshal event",
					logger.Error(err))
				continue
			}

			// Broadcast to all clients
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.messages <- data:
				default:
					// Client buffer full, skip
					h.logger.Warn("Client message buffer full, skipping",
						logger.String("client_id", client.ID))
				}
			}
			h.mu.RUnlock()

		case <-ctx.Done():
			h.logger.Info("WebSocket hub shutting down")
			// Close all client connections
			h.mu.Lock()
			for client := range h.clients {
				close(client.messages)
			}
			h.clients = make(map[*Client]bool)
			h.mu.Unlock()
			return
		}
	}
}

// Broadcast sends an event to all connected clients
func (h *WebSocketHub) Broadcast(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	select {
	case h.broadcast <- event:
	default:
		h.logger.Warn("Broadcast channel full, dropping event",
			logger.String("event_type", event.Type))
	}
}

// Handler returns an HTTP handler for WebSocket connections
func (h *WebSocketHub) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For now, return a simple response indicating WebSocket endpoint
		// In a full implementation, this would upgrade the connection
		// using gorilla/websocket or similar library
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"endpoint": "websocket",
			"status":   "available",
		})
	})
}

// GetClientCount returns the number of connected clients
func (h *WebSocketHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
