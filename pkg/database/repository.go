package database

import (
	"time"

	"gorm.io/gorm"
)

// TransmissionRepository handles transmission database operations
type TransmissionRepository struct {
	db *gorm.DB
}

// NewTransmissionRepository creates a new transmission repository
func NewTransmissionRepository(db *gorm.DB) *TransmissionRepository {
	return &TransmissionRepository{db: db}
}

// Create adds a new transmission record
func (r *TransmissionRepository) Create(tx *Transmission) error {
	return r.db.Create(tx).Error
}

// GetRecent retrieves the most recent N transmissions
func (r *TransmissionRepository) GetRecent(limit int) ([]Transmission, error) {
	var transmissions []Transmission
	err := r.db.Order("start_time DESC").Limit(limit).Find(&transmissions).Error
	return transmissions, err
}

// GetRecentPaginated retrieves transmissions with pagination
func (r *TransmissionRepository) GetRecentPaginated(page, perPage int) ([]Transmission, int64, error) {
	var transmissions []Transmission
	var total int64

	// Count total records
	if err := r.db.Model(&Transmission{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (page - 1) * perPage
	err := r.db.Order("start_time DESC").
		Offset(offset).
		Limit(perPage).
		Find(&transmissions).Error

	return transmissions, total, err
}

// GetByRadioID retrieves transmissions for a specific radio
func (r *TransmissionRepository) GetByRadioID(radioID uint32, limit int) ([]Transmission, error) {
	var transmissions []Transmission
	err := r.db.Where("radio_id = ?", radioID).
		Order("start_time DESC").
		Limit(limit).
		Find(&transmissions).Error
	return transmissions, err
}

// GetByTalkgroup retrieves transmissions for a specific talkgroup
func (r *TransmissionRepository) GetByTalkgroup(tgID uint32, limit int) ([]Transmission, error) {
	var transmissions []Transmission
	err := r.db.Where("talkgroup_id = ?", tgID).
		Order("start_time DESC").
		Limit(limit).
		Find(&transmissions).Error
	return transmissions, err
}

// GetByTimeRange retrieves transmissions within a time range
func (r *TransmissionRepository) GetByTimeRange(start, end time.Time, limit int) ([]Transmission, error) {
	var transmissions []Transmission
	err := r.db.Where("start_time BETWEEN ? AND ?", start, end).
		Order("start_time DESC").
		Limit(limit).
		Find(&transmissions).Error
	return transmissions, err
}

// DeleteOlderThan deletes transmissions older than the specified time
func (r *TransmissionRepository) DeleteOlderThan(before time.Time) (int64, error) {
	result := r.db.Where("start_time < ?", before).Delete(&Transmission{})
	return result.RowsAffected, result.Error
}

// GetActiveStreamIDs retrieves stream IDs that are currently active (within last N seconds)
func (r *TransmissionRepository) GetActiveStreamIDs(withinSeconds int) ([]uint32, error) {
	var streamIDs []uint32
	cutoff := time.Now().Add(-time.Duration(withinSeconds) * time.Second)

	err := r.db.Model(&Transmission{}).
		Where("end_time > ?", cutoff).
		Distinct("stream_id").
		Pluck("stream_id", &streamIDs).Error

	return streamIDs, err
}
