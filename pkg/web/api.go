package web

import (
	"encoding/json"
	"net/http"

	"github.com/dbehnke/dmr-nexus/pkg/bridge"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/dbehnke/dmr-nexus/pkg/peer"
)

// API handles REST API endpoints
type API struct {
	logger *logger.Logger
	peers  *peer.PeerManager
	router *bridge.Router
}

// NewAPI creates a new API instance
func NewAPI(log *logger.Logger) *API { return &API{logger: log} }

// SetDeps provides runtime dependencies to the API after construction
func (a *API) SetDeps(pm *peer.PeerManager, r *bridge.Router) {
	a.peers = pm
	a.router = r
}

// PeerDTO is a lightweight response for peer info
type PeerDTO struct {
	ID          uint32   `json:"id"`
	Callsign    string   `json:"callsign"`
	Address     string   `json:"address"`
	State       string   `json:"state"`
	Location    string   `json:"location"`
	ConnectedAt int64    `json:"connected_at"`
	LastHeard   int64    `json:"last_heard"`
	PacketsRx   uint64   `json:"packets_rx"`
	BytesRx     uint64   `json:"bytes_rx"`
	PacketsTx   uint64   `json:"packets_tx"`
	BytesTx     uint64   `json:"bytes_tx"`
	TS1         []uint32 `json:"ts1,omitempty"`
	TS2         []uint32 `json:"ts2,omitempty"`
}

// BridgeDTO is a lightweight response for bridge rules
type BridgeDTO struct {
	Name  string          `json:"name"`
	Rules []BridgeRuleDTO `json:"rules"`
}

type BridgeRuleDTO struct {
	System   string `json:"system"`
	TGID     int    `json:"tgid"`
	Timeslot int    `json:"timeslot"`
	Active   bool   `json:"active"`
}

// DynamicBridgeDTO is a lightweight response for dynamic bridges
type DynamicBridgeDTO struct {
	TGID         uint32   `json:"tgid"`
	Timeslot     int      `json:"timeslot"`
	CreatedAt    int64    `json:"created_at"`
	LastActivity int64    `json:"last_activity"`
	Subscribers  []uint32 `json:"subscribers"`
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
	// If manager not wired, return empty list for forward compatibility
	if a.peers == nil {
		if err := json.NewEncoder(w).Encode([]PeerDTO{}); err != nil {
			a.logger.Error("Failed to encode peers response", logger.Error(err))
		}
		return
	}

	// Build DTOs from snapshots
	list := make([]PeerDTO, 0)
	for _, p := range a.peers.GetAllPeers() {
		snap := p.Snapshot(true)
		list = append(list, PeerDTO{
			ID:          snap.ID,
			Callsign:    snap.Callsign,
			Address:     snap.Address,
			State:       snap.State,
			Location:    snap.Location,
			ConnectedAt: snap.ConnectedAt.Unix(),
			LastHeard:   snap.LastHeard.Unix(),
			PacketsRx:   snap.PacketsRx,
			BytesRx:     snap.BytesRx,
			PacketsTx:   snap.PacketsTx,
			BytesTx:     snap.BytesTx,
			TS1:         snap.Subscriptions.TS1,
			TS2:         snap.Subscriptions.TS2,
		})
	}
	if err := json.NewEncoder(w).Encode(list); err != nil {
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

	response := map[string]interface{}{
		"static":  []BridgeDTO{},
		"dynamic": []DynamicBridgeDTO{},
	}

	if a.router == nil {
		if err := json.NewEncoder(w).Encode(response); err != nil {
			a.logger.Error("Failed to encode bridges response", logger.Error(err))
		}
		return
	}

	// Build DTOs from static router bridges using snapshots
	staticBridges := make([]BridgeDTO, 0)
	for _, br := range a.router.GetActiveBridges() {
		snap := br.Snapshot()
		dto := BridgeDTO{Name: snap.Name, Rules: make([]BridgeRuleDTO, 0, len(snap.Rules))}
		for _, rs := range snap.Rules {
			dto.Rules = append(dto.Rules, BridgeRuleDTO{
				System:   rs.System,
				TGID:     rs.TGID,
				Timeslot: rs.Timeslot,
				Active:   rs.Active,
			})
		}
		staticBridges = append(staticBridges, dto)
	}
	response["static"] = staticBridges

	// Build DTOs from dynamic bridges
	dynamicBridges := make([]DynamicBridgeDTO, 0)
	for _, db := range a.router.GetAllDynamicBridges() {
		// Derive subscribers from current peer subscriptions to avoid stale data
		subscribers := make([]uint32, 0)
		if a.peers != nil {
			for _, p := range a.peers.GetAllPeers() {
				if p.GetState().String() != "connected" {
					continue
				}
				if p.Subscriptions != nil && p.Subscriptions.IsSubscribedToTalkgroup(db.TGID) {
					subscribers = append(subscribers, p.ID)
				}
			}
		}

		dynamicBridges = append(dynamicBridges, DynamicBridgeDTO{
			TGID:         db.TGID,
			Timeslot:     db.Timeslot,
			CreatedAt:    db.CreatedAt.Unix(),
			LastActivity: db.LastActivity.Unix(),
			Subscribers:  subscribers,
		})
	}
	response["dynamic"] = dynamicBridges

	if err := json.NewEncoder(w).Encode(response); err != nil {
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
