package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/bridge"
	"github.com/dbehnke/dmr-nexus/pkg/database"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/dbehnke/dmr-nexus/pkg/peer"
)

// API handles REST API endpoints
type API struct {
	logger    *logger.Logger
	peers     *peer.PeerManager
	router    *bridge.Router
	txRepo    *database.TransmissionRepository
	streamMap map[uint32]*streamActivity // Track active streams
}

// streamActivity tracks active transmission metadata
type streamActivity struct {
	streamID    uint32
	radioID     uint32
	talkgroupID uint32
	timeslot    int
	repeaterID  uint32
	startTime   time.Time
	lastSeen    time.Time
	packetCount int
}

// NewAPI creates a new API instance
func NewAPI(log *logger.Logger) *API {
	return &API{
		logger:    log,
		streamMap: make(map[uint32]*streamActivity),
	}
}

// SetDeps provides runtime dependencies to the API after construction
func (a *API) SetDeps(pm *peer.PeerManager, r *bridge.Router) {
	a.peers = pm
	a.router = r
}

// SetTransmissionRepo sets the transmission repository
func (a *API) SetTransmissionRepo(repo *database.TransmissionRepository) {
	a.txRepo = repo
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
	Active       bool     `json:"active"` // Whether someone is currently talking
}

// TransmissionDTO is a lightweight response for transmissions
type TransmissionDTO struct {
	ID          uint    `json:"id"`
	RadioID     uint32  `json:"radio_id"`
	TalkgroupID uint32  `json:"talkgroup_id"`
	Timeslot    int     `json:"timeslot"`
	Duration    float64 `json:"duration"`
	StartTime   int64   `json:"start_time"`
	EndTime     int64   `json:"end_time"`
	RepeaterID  uint32  `json:"repeater_id"`
	PacketCount int     `json:"packet_count"`
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

		// Check if this bridge is active (recent activity within 5 seconds)
		active := time.Since(db.LastActivity) < 5*time.Second

		dynamicBridges = append(dynamicBridges, DynamicBridgeDTO{
			TGID:         db.TGID,
			Timeslot:     db.Timeslot,
			CreatedAt:    db.CreatedAt.Unix(),
			LastActivity: db.LastActivity.Unix(),
			Subscribers:  subscribers,
			Active:       active,
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

// HandleTransmissions handles the /api/transmissions endpoint
func (a *API) HandleTransmissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// If no transmission repo, return empty list
	if a.txRepo == nil {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"transmissions": []TransmissionDTO{},
			"total":         0,
			"page":          1,
			"per_page":      50,
		}); err != nil {
			a.logger.Error("Failed to encode transmissions response", logger.Error(err))
		}
		return
	}

	// Parse pagination parameters
	page := 1
	perPage := 50

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	// Get transmissions from database
	transmissions, total, err := a.txRepo.GetRecentPaginated(page, perPage)
	if err != nil {
		a.logger.Error("Failed to get transmissions", logger.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to DTOs
	dtos := make([]TransmissionDTO, 0, len(transmissions))
	for _, tx := range transmissions {
		dtos = append(dtos, TransmissionDTO{
			ID:          tx.ID,
			RadioID:     tx.RadioID,
			TalkgroupID: tx.TalkgroupID,
			Timeslot:    tx.Timeslot,
			Duration:    tx.Duration,
			StartTime:   tx.StartTime.Unix(),
			EndTime:     tx.EndTime.Unix(),
			RepeaterID:  tx.RepeaterID,
			PacketCount: tx.PacketCount,
		})
	}

	response := map[string]interface{}{
		"transmissions": dtos,
		"total":         total,
		"page":          page,
		"per_page":      perPage,
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		a.logger.Error("Failed to encode transmissions response", logger.Error(err))
	}
}

