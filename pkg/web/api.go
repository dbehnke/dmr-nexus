package web

import (
	"encoding/json"
	"net/http"

	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

// API handles REST API endpoints
type API struct {
	logger *logger.Logger
}

// NewAPI creates a new API instance
func NewAPI(log *logger.Logger) *API {
	return &API{
		logger: log,
	}
}

// HandleStatus handles the /api/status endpoint
func (a *API) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":  "running",
		"service": "dmr-nexus",
		"version": "dev",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		a.logger.Error("Failed to encode status response", logger.Error(err))
	}
}

// HandlePeers handles the /api/peers endpoint
func (a *API) HandlePeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Return empty array for now - will be populated with actual peer data
	peers := []interface{}{}
	if err := json.NewEncoder(w).Encode(peers); err != nil {
		a.logger.Error("Failed to encode peers response", logger.Error(err))
	}
}

// HandleBridges handles the /api/bridges endpoint
func (a *API) HandleBridges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Return empty array for now - will be populated with actual bridge data
	bridges := []interface{}{}
	if err := json.NewEncoder(w).Encode(bridges); err != nil {
		a.logger.Error("Failed to encode bridges response", logger.Error(err))
	}
}

// HandleActivity handles the /api/activity endpoint
func (a *API) HandleActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Return empty array for now - will be populated with actual activity data
	activity := []interface{}{}
	if err := json.NewEncoder(w).Encode(activity); err != nil {
		a.logger.Error("Failed to encode activity response", logger.Error(err))
	}
}
