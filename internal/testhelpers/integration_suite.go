package testhelpers

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/config"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

// IntegrationSuite provides infrastructure for integration tests
type IntegrationSuite struct {
	T          *testing.T
	Config     *config.Config
	Logger     *logger.Logger
	Ctx        context.Context
	Cancel     context.CancelFunc
	MockPeers  []*MockPeer
	TestServer *TestServer
}

// TestServer represents a test DMR server
type TestServer struct {
	Port   int
	Addr   string
	cancel context.CancelFunc
}

// NewIntegrationSuite creates a new integration test suite
func NewIntegrationSuite(t *testing.T) *IntegrationSuite {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	log := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
	})

	return &IntegrationSuite{
		T:         t,
		Logger:    log,
		Ctx:       ctx,
		Cancel:    cancel,
		MockPeers: make([]*MockPeer, 0),
	}
}

// CreateMockPeer creates a new mock peer and adds it to the suite
func (s *IntegrationSuite) CreateMockPeer(peerID uint32, passphrase string, callsign string) *MockPeer {
	peer := NewMockPeer(peerID, passphrase, callsign)
	s.MockPeers = append(s.MockPeers, peer)
	return peer
}

// GetFreePort gets a free port for testing
func (s *IntegrationSuite) GetFreePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		s.T.Fatal(err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		s.T.Fatal(err)
	}
	defer func() { _ = listener.Close() }()

	return listener.Addr().(*net.TCPAddr).Port
}

// StartTestServer starts a test DMR server
func (s *IntegrationSuite) StartTestServer(cfg *config.Config) *TestServer {
	_, cancel := context.WithCancel(s.Ctx)

	port := s.GetFreePort()
	addr := fmt.Sprintf("localhost:%d", port)

	server := &TestServer{
		Port:   port,
		Addr:   addr,
		cancel: cancel,
	}

	s.TestServer = server

	// TODO: Start actual DMR server when fully integrated
	// For now, this is a placeholder that allows tests to be written

	return server
}

// StopTestServer stops the test server
func (s *IntegrationSuite) StopTestServer() {
	if s.TestServer != nil && s.TestServer.cancel != nil {
		s.TestServer.cancel()
	}
}

// Cleanup cleans up resources
func (s *IntegrationSuite) Cleanup() {
	// Close all mock peers
	for _, peer := range s.MockPeers {
		_ = peer.Close()
	}

	// Stop test server
	s.StopTestServer()

	// Cancel context
	s.Cancel()
}

// WaitFor waits for a condition to be true
func (s *IntegrationSuite) WaitFor(condition func() bool, timeout time.Duration, message string) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.T.Logf("WaitFor timeout: %s", message)
	return false
}

// AssertEventually asserts that a condition becomes true within timeout
func (s *IntegrationSuite) AssertEventually(condition func() bool, timeout time.Duration, message string) {
	if !s.WaitFor(condition, timeout, message) {
		s.T.Errorf("Assertion failed: %s", message)
	}
}

// CreateDefaultConfig creates a default test configuration
func CreateDefaultConfig() *config.Config {
	return &config.Config{
		Global: config.GlobalConfig{
			PingTime:  5,
			MaxMissed: 3,
			UseACL:    true,
			RegACL:    "PERMIT:ALL",
			SubACL:    "DENY:1",
			TG1ACL:    "PERMIT:ALL",
			TG2ACL:    "PERMIT:ALL",
		},
		Server: config.ServerConfig{
			Name:        "Test Server",
			Description: "Integration Test Server",
		},
		Web: config.WebConfig{
			Enabled: false,
		},
		MQTT: config.MQTTConfig{
			Enabled: false,
		},
		Metrics: config.MetricsConfig{
			Enabled: false,
		},
		Systems: make(map[string]config.SystemConfig),
		Bridges: make(map[string][]config.BridgeRule),
	}
}
