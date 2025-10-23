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
	logger   *logger.Logger
	peers    *peer.PeerManager
	router   *bridge.Router
	txRepo   *database.TransmissionRepository
	userRepo *database.DMRUserRepository
}

// streamActivity tracks active transmission metadata
// streamActivity previously tracked active transmission metadata. It was
// removed because the API currently uses the Transmission repository and
// router/dynamic bridge activity timestamps. Reintroduce if active stream
// tracking is required in the future.

// NewAPI creates a new API instance
func NewAPI(log *logger.Logger) *API {
	return &API{
		logger: log,
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

// SetUserRepo sets the user repository
func (a *API) SetUserRepo(repo *database.DMRUserRepository) {
	a.userRepo = repo
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

// SubscriberInfo represents a subscriber and which timeslot(s) they're subscribed on
type SubscriberInfo struct {
	PeerID   uint32 `json:"peer_id"`
	Timeslot int    `json:"timeslot"` // 1=TS1 only, 2=TS2 only, 3=both timeslots
}

// DynamicBridgeDTO is a lightweight response for dynamic bridges
// Bridges are timeslot-agnostic - they show all subscribers across both timeslots
type DynamicBridgeDTO struct {
	TGID          uint32           `json:"tgid"`
	CreatedAt     int64            `json:"created_at"`
	LastActivity  int64            `json:"last_activity"`
	Subscribers   []SubscriberInfo `json:"subscribers"`
	Active        bool             `json:"active"`          // Whether someone is currently talking
	ActiveRadioID uint32           `json:"active_radio_id"` // Radio ID currently transmitting (0 if none)
	// User info for active radio (if available)
	ActiveCallsign  string `json:"active_callsign,omitempty"`
	ActiveFirstName string `json:"active_first_name,omitempty"`
	ActiveLastName  string `json:"active_last_name,omitempty"`
	ActiveLocation  string `json:"active_location,omitempty"`
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
	// User info (if available)
	Callsign string `json:"callsign,omitempty"`
}

// HandleStatus handles the /api/status endpoint
func (a *API) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	versionStr, commit, buildTime := GetVersionInfo()

	response := map[string]interface{}{
		"status":     "running",
		"service":    "dmr-nexus",
		"version":    versionStr,
		"commit":     commit,
		"build_time": buildTime,
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
		// Check which timeslot(s) each subscriber is on
		subscribers := make([]SubscriberInfo, 0)
		if a.peers != nil {
			for _, p := range a.peers.GetAllPeers() {
				if p.GetState().String() != "connected" {
					continue
				}
				if p.Subscriptions == nil {
					continue
				}

				// Check if subscribed on TS1, TS2, or both
				ts1 := p.Subscriptions.IsSubscribed(db.TGID, 1)
				ts2 := p.Subscriptions.IsSubscribed(db.TGID, 2)

				if ts1 || ts2 {
					timeslot := 0
					if ts1 && ts2 {
						timeslot = 3 // Both timeslots
					} else if ts1 {
						timeslot = 1 // TS1 only
					} else {
						timeslot = 2 // TS2 only
					}

					subscribers = append(subscribers, SubscriberInfo{
						PeerID:   p.ID,
						Timeslot: timeslot,
					})
				}
			}
		}

		// Check if this bridge is active (recent activity within 5 seconds)
		active := time.Since(db.LastActivity) < 5*time.Second

		dto := DynamicBridgeDTO{
			TGID:          db.TGID,
			CreatedAt:     db.CreatedAt.Unix(),
			LastActivity:  db.LastActivity.Unix(),
			Subscribers:   subscribers,
			Active:        active,
			ActiveRadioID: db.ActiveRadioID,
		}

		// If active and we have a user repo, look up user info for active radio
		if active && db.ActiveRadioID != 0 && a.userRepo != nil {
			if user, err := a.userRepo.GetByRadioID(db.ActiveRadioID); err == nil {
				dto.ActiveCallsign = user.Callsign
				dto.ActiveFirstName = user.FirstName
				dto.ActiveLastName = user.LastName
				dto.ActiveLocation = user.Location()
			}
		}

		dynamicBridges = append(dynamicBridges, dto)
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
		dto := TransmissionDTO{
			ID:          tx.ID,
			RadioID:     tx.RadioID,
			TalkgroupID: tx.TalkgroupID,
			Timeslot:    tx.Timeslot,
			Duration:    tx.Duration,
			StartTime:   tx.StartTime.Unix(),
			EndTime:     tx.EndTime.Unix(),
			RepeaterID:  tx.RepeaterID,
			PacketCount: tx.PacketCount,
		}

		// Look up callsign if user repo is available
		if a.userRepo != nil {
			if user, err := a.userRepo.GetByRadioID(tx.RadioID); err == nil {
				dto.Callsign = user.Callsign
			}
		}

		dtos = append(dtos, dto)
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

// UserDTO is a lightweight response for user info
type UserDTO struct {
	RadioID   uint32 `json:"radio_id"`
	Callsign  string `json:"callsign"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	City      string `json:"city"`
	State     string `json:"state"`
	Country   string `json:"country"`
	Location  string `json:"location"`
}

// HandleUserLookup handles the /api/user/:radio_id endpoint
func (a *API) HandleUserLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract radio ID from path
	path := r.URL.Path
	radioIDStr := path[len("/api/user/"):]
	if radioIDStr == "" {
		http.Error(w, "Radio ID required", http.StatusBadRequest)
		return
	}

	radioID64, err := strconv.ParseUint(radioIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid radio ID", http.StatusBadRequest)
		return
	}
	radioID := uint32(radioID64)

	w.Header().Set("Content-Type", "application/json")

	// If no user repo, return 404
	if a.userRepo == nil {
		http.Error(w, "User lookup not available", http.StatusServiceUnavailable)
		return
	}

	// Look up user
	user, err := a.userRepo.GetByRadioID(radioID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	dto := UserDTO{
		RadioID:   user.RadioID,
		Callsign:  user.Callsign,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		City:      user.City,
		State:     user.State,
		Country:   user.Country,
		Location:  user.Location(),
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(dto); err != nil {
		a.logger.Error("Failed to encode user response", logger.Error(err))
	}
}
