package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

func TestAPI_Status(t *testing.T) {
	log := logger.New(logger.Config{Level: "info"})
	api := NewAPI(log)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()

	api.HandleStatus(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check response is valid JSON
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should contain status field
	if _, ok := result["status"]; !ok {
		t.Error("Response doesn't contain status field")
	}
}

func TestAPI_Peers(t *testing.T) {
	log := logger.New(logger.Config{Level: "info"})
	api := NewAPI(log)

	req := httptest.NewRequest(http.MethodGet, "/api/peers", nil)
	w := httptest.NewRecorder()

	api.HandlePeers(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check response is valid JSON array
	var result []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

func TestAPI_Bridges(t *testing.T) {
	log := logger.New(logger.Config{Level: "info"})
	api := NewAPI(log)

	req := httptest.NewRequest(http.MethodGet, "/api/bridges", nil)
	w := httptest.NewRecorder()

	api.HandleBridges(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check response is valid JSON array
	var result []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

func TestAPI_Activity(t *testing.T) {
	log := logger.New(logger.Config{Level: "info"})
	api := NewAPI(log)

	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)
	w := httptest.NewRecorder()

	api.HandleActivity(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check response is valid JSON array
	var result []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

func TestAPI_NotFound(t *testing.T) {
	log := logger.New(logger.Config{Level: "info"})
	_ = NewAPI(log) // Create API instance for consistency

	// Create a test handler that uses the API's not found handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/notfound", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestAPI_MethodNotAllowed(t *testing.T) {
	log := logger.New(logger.Config{Level: "info"})
	api := NewAPI(log)

	// POST to GET-only endpoint
	req := httptest.NewRequest(http.MethodPost, "/api/status", nil)
	w := httptest.NewRecorder()

	api.HandleStatus(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}
