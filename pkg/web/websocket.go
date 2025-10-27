package web

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/gorilla/websocket"
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
	conn     *websocket.Conn
	messages chan []byte
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
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "websocket upgrade failed", http.StatusBadRequest)
			return
		}
		client := &Client{ID: r.RemoteAddr, conn: conn, messages: make(chan []byte, 256)}
		h.register <- client

		// Reader goroutine: drain read to detect close
		go func() {
			defer func() {
				h.unregister <- client
				_ = client.conn.Close()
			}()
			client.conn.SetReadLimit(1024)
			for {
				if _, _, err := client.conn.ReadMessage(); err != nil {
					return
				}
			}
		}()

		// Writer loop
		go func() {
			for msg := range client.messages {
				_ = client.conn.WriteMessage(websocket.TextMessage, msg)
			}
		}()
	})
}

// GetClientCount returns the number of connected clients
func (h *WebSocketHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Helper broadcasters for common event types
func (h *WebSocketHub) BroadcastPeerConnected(id uint32, callsign string, addr string) {
	h.Broadcast(Event{
		Type:      "peer_connected",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"id":       id,
			"callsign": callsign,
			"addr":     addr,
		},
	})
}

func (h *WebSocketHub) BroadcastPeerDisconnected(id uint32) {
	h.Broadcast(Event{
		Type:      "peer_disconnected",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"id": id,
		},
	})
}

// BroadcastStatusUpdate broadcasts a status update to all clients
func (h *WebSocketHub) BroadcastStatusUpdate(status string, version string) {
	h.Broadcast(Event{
		Type:      "status_update",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status":  status,
			"version": version,
		},
	})
}

// BroadcastPeersUpdate broadcasts peer list update to all clients
func (h *WebSocketHub) BroadcastPeersUpdate(peers interface{}) {
	h.Broadcast(Event{
		Type:      "peers_update",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"peers": peers,
		},
	})
}

// BroadcastBridgesUpdate broadcasts bridge list update to all clients
func (h *WebSocketHub) BroadcastBridgesUpdate(bridges interface{}) {
	h.Broadcast(Event{
		Type:      "bridges_update",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"bridges": bridges,
		},
	})
}

// BroadcastTransmissionsUpdate broadcasts transmission update to all clients
func (h *WebSocketHub) BroadcastTransmissionsUpdate(transmissions interface{}) {
	h.Broadcast(Event{
		Type:      "transmissions_update",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"transmissions": transmissions,
		},
	})
}
