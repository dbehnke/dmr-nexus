//go:build integration
// +build integration

package testhelpers

import (
	"testing"
	"time"
)

// TestIntegrationSuite_Basic tests basic integration suite functionality
func TestIntegrationSuite_Basic(t *testing.T) {
	suite := NewIntegrationSuite(t)
	defer suite.Cleanup()

	if suite.Logger == nil {
		t.Error("Expected logger to be initialized")
	}

	if suite.Ctx == nil {
		t.Error("Expected context to be initialized")
	}
}

// TestIntegrationSuite_MockPeer tests creating mock peers
func TestIntegrationSuite_MockPeer(t *testing.T) {
	suite := NewIntegrationSuite(t)
	defer suite.Cleanup()

	peer := suite.CreateMockPeer(312000, "password", "W1ABC")
	if peer == nil {
		t.Fatal("Expected non-nil peer")
	}

	if peer.PeerID != 312000 {
		t.Errorf("Expected peer ID 312000, got %d", peer.PeerID)
	}

	if peer.Callsign != "W1ABC" {
		t.Errorf("Expected callsign W1ABC, got %s", peer.Callsign)
	}

	if len(suite.MockPeers) != 1 {
		t.Errorf("Expected 1 mock peer, got %d", len(suite.MockPeers))
	}
}

// TestIntegrationSuite_WaitFor tests the WaitFor helper
func TestIntegrationSuite_WaitFor(t *testing.T) {
	suite := NewIntegrationSuite(t)
	defer suite.Cleanup()

	counter := 0
	condition := func() bool {
		counter++
		return counter >= 5
	}

	result := suite.WaitFor(condition, 1*time.Second, "counter >= 5")
	if !result {
		t.Error("Expected WaitFor to succeed")
	}

	if counter < 5 {
		t.Errorf("Expected counter >= 5, got %d", counter)
	}
}

// TestIntegrationSuite_WaitForTimeout tests WaitFor timeout
func TestIntegrationSuite_WaitForTimeout(t *testing.T) {
	suite := NewIntegrationSuite(t)
	defer suite.Cleanup()

	condition := func() bool {
		return false
	}

	result := suite.WaitFor(condition, 100*time.Millisecond, "always false")
	if result {
		t.Error("Expected WaitFor to timeout")
	}
}

// TestIntegrationSuite_GetFreePort tests getting a free port
func TestIntegrationSuite_GetFreePort(t *testing.T) {
	suite := NewIntegrationSuite(t)
	defer suite.Cleanup()

	port := suite.GetFreePort()
	if port <= 0 || port > 65535 {
		t.Errorf("Invalid port number: %d", port)
	}
}

// TestDefaultConfig tests creating a default configuration
func TestDefaultConfig(t *testing.T) {
	cfg := CreateDefaultConfig()

	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}

	if cfg.Global.PingTime != 5 {
		t.Errorf("Expected ping time 5, got %d", cfg.Global.PingTime)
	}

	if cfg.Server.Name != "Test Server" {
		t.Errorf("Expected server name 'Test Server', got %s", cfg.Server.Name)
	}
}
