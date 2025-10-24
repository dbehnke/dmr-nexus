package web

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/database"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/dbehnke/dmr-nexus/pkg/peer"
)

func TestMaskIPAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "IPv4 with port",
			input:    "162.1.2.3:8080",
			expected: "162.1.*.*:8080",
		},
		{
			name:     "IPv4 without port",
			input:    "192.168.1.1",
			expected: "192.168.*.*",
		},
		{
			name:     "IPv4 with different octets",
			input:    "67.220.71.98:36098",
			expected: "67.220.*.*:36098",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "IPv6 with brackets and port",
			input:    "[::1]:8080",
			expected: "[::1]:8080",
		},
		{
			name:     "IPv6 without brackets",
			input:    "2001:db8::1",
			expected: "2001:db8::1",
		},
		{
			name:     "Invalid format",
			input:    "not-an-ip",
			expected: "not-an-ip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskIPAddress(tt.input)
			if result != tt.expected {
				t.Errorf("maskIPAddress(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDynamicBridgeSubscribers_AAA(t *testing.T) {
	// Arrange
	pm := peer.NewPeerManager()
	addr1 := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 10001}
	addr2 := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 10002}
	p1 := pm.AddPeer(1001, addr1)
	p2 := pm.AddPeer(1002, addr2)

	// Use exported Subscription API
	p1.GetSubscriptions().AddDynamic(7000, 1)
	p2.GetSubscriptions().AddDynamic(7000, 2)

	// Act
	var ts1, ts2 []uint32
	for _, p := range pm.GetAllPeers() {
		if p.HasSubscription(7000, 1) {
			ts1 = append(ts1, p.ID)
		}
		if p.HasSubscription(7000, 2) {
			ts2 = append(ts2, p.ID)
		}
	}

	// Assert
	if len(ts1) != 1 || ts1[0] != 1001 {
		t.Errorf("TS1 subscribers incorrect: %v", ts1)
	}
	if len(ts2) != 1 || ts2[0] != 1002 {
		t.Errorf("TS2 subscribers incorrect: %v", ts2)
	}
}

func TestDashboardBridgeCount_AAA(t *testing.T) {
	// Arrange
	type Bridge struct {
		ID      int
		Dynamic bool
	}
	bridges := []Bridge{
		{ID: 1, Dynamic: false},
		{ID: 2, Dynamic: true},
		{ID: 3, Dynamic: true},
	}

	// Act
	staticCount := 0
	dynamicCount := 0
	for _, b := range bridges {
		if b.Dynamic {
			dynamicCount++
		} else {
			staticCount++
		}
	}
	activeCount := staticCount + dynamicCount

	// Assert
	if staticCount != 1 {
		t.Errorf("Expected 1 static bridge, got %d", staticCount)
	}
	if dynamicCount != 2 {
		t.Errorf("Expected 2 dynamic bridges, got %d", dynamicCount)
	}
	if activeCount != 3 {
		t.Errorf("Expected 3 active bridges, got %d", activeCount)
	}
}

func TestHandleTransmissions_NoRepo(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	api := NewAPI(log)

	req := httptest.NewRequest("GET", "/api/transmissions", nil)
	w := httptest.NewRecorder()

	api.HandleTransmissions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if total, ok := response["total"].(float64); !ok || total != 0 {
		t.Errorf("Expected total 0, got %v", response["total"])
	}
}

func TestHandleTransmissions_WithData(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_api_transmissions.db"
	defer func() {
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			t.Fatalf("failed to remove db file %s: %v", dbPath, err)
		}
	}()

	db, err := database.NewDB(database.Config{Path: dbPath}, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()

	repo := database.NewTransmissionRepository(db.GetDB())

	// Create test transmissions
	now := time.Now()
	for i := 0; i < 3; i++ {
		tx := &database.Transmission{
			RadioID:     uint32(1234560 + i),
			TalkgroupID: 91,
			Timeslot:    1,
			Duration:    float64(i + 1),
			StreamID:    uint32(1000 + i),
			StartTime:   now.Add(time.Duration(i) * time.Minute),
			EndTime:     now.Add(time.Duration(i)*time.Minute + time.Duration(i+1)*time.Second),
			RepeaterID:  3001,
			PacketCount: 10 + i,
		}
		if err := repo.Create(tx); err != nil {
			t.Fatalf("Failed to create transmission: %v", err)
		}
	}

	// Create API with repo
	api := NewAPI(log)
	api.SetTransmissionRepo(repo)

	req := httptest.NewRequest("GET", "/api/transmissions?page=1&per_page=2", nil)
	w := httptest.NewRecorder()

	api.HandleTransmissions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if total, ok := response["total"].(float64); !ok || total != 3 {
		t.Errorf("Expected total 3, got %v", response["total"])
	}

	if page, ok := response["page"].(float64); !ok || page != 1 {
		t.Errorf("Expected page 1, got %v", response["page"])
	}

	if perPage, ok := response["per_page"].(float64); !ok || perPage != 2 {
		t.Errorf("Expected per_page 2, got %v", response["per_page"])
	}

	transmissions, ok := response["transmissions"].([]interface{})
	if !ok {
		t.Fatalf("Expected transmissions array")
	}

	if len(transmissions) != 2 {
		t.Errorf("Expected 2 transmissions on first page, got %d", len(transmissions))
	}
}

func TestHandleTransmissions_MethodNotAllowed(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	api := NewAPI(log)

	req := httptest.NewRequest("POST", "/api/transmissions", nil)
	w := httptest.NewRecorder()

	api.HandleTransmissions(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandlePeers_MaskedIPAddress(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	api := NewAPI(log)

	// Create peer manager with test peers
	pm := peer.NewPeerManager()
	addr1 := &net.UDPAddr{IP: net.ParseIP("162.1.2.3"), Port: 8080}
	addr2 := &net.UDPAddr{IP: net.ParseIP("67.220.71.98"), Port: 36098}

	p1 := pm.AddPeer(320044901, addr1)
	p1.SetConnected()
	p1.Callsign = "WC8MI"
	p1.Location = "Nowhere"

	p2 := pm.AddPeer(123456789, addr2)
	p2.SetConnected()
	p2.Callsign = "TEST"

	api.SetDeps(pm, nil)

	// Make request
	req := httptest.NewRequest("GET", "/api/peers", nil)
	w := httptest.NewRecorder()

	api.HandlePeers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var peers []PeerDTO
	if err := json.NewDecoder(w.Body).Decode(&peers); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(peers) != 2 {
		t.Fatalf("Expected 2 peers, got %d", len(peers))
	}

	// Verify IP addresses are masked
	for _, p := range peers {
		switch p.ID {
		case 320044901:
			if p.Address != "162.1.*.*:8080" {
				t.Errorf("Expected masked address '162.1.*.*:8080', got '%s'", p.Address)
			}
			if p.Callsign != "WC8MI" {
				t.Errorf("Expected callsign 'WC8MI', got '%s'", p.Callsign)
			}
		case 123456789:
			if p.Address != "67.220.*.*:36098" {
				t.Errorf("Expected masked address '67.220.*.*:36098', got '%s'", p.Address)
			}
		}
	}
}
