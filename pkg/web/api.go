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

	json.NewEncoder(w).Encode(response)
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
	json.NewEncoder(w).Encode(peers)
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
	json.NewEncoder(w).Encode(bridges)
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
	json.NewEncoder(w).Encode(activity)
}
