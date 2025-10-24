package database

import (
	"time"

	"gorm.io/gorm"
)

// Transmission represents a DMR transmission record
type Transmission struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	RadioID     uint32    `gorm:"index;not null" json:"radio_id"`
	TalkgroupID uint32    `gorm:"index;not null" json:"talkgroup_id"`
	Timeslot    int       `gorm:"not null" json:"timeslot"`
	Duration    float64   `gorm:"not null" json:"duration"` // Duration in seconds
	StreamID    uint32    `gorm:"index" json:"stream_id"`
	StartTime   time.Time `gorm:"index;not null" json:"start_time"`
	EndTime     time.Time `gorm:"not null" json:"end_time"`
	RepeaterID  uint32    `gorm:"index" json:"repeater_id"`
	PacketCount int       `gorm:"default:0" json:"packet_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableName specifies the table name for Transmission
func (Transmission) TableName() string {
	return "transmissions"
}

// BeforeCreate hook to ensure StartTime and EndTime are set
func (t *Transmission) BeforeCreate(tx *gorm.DB) error {
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	if t.StartTime.IsZero() {
		t.StartTime = time.Now()
	}
	if t.EndTime.IsZero() {
		t.EndTime = time.Now()
	}
	return nil
}

// DMRUser represents a DMR user from the RadioID database
type DMRUser struct {
	RadioID   uint32    `gorm:"primarykey;not null" json:"radio_id"`
	Callsign  string    `gorm:"index;size:20" json:"callsign"`
	FirstName string    `gorm:"size:50" json:"first_name"`
	LastName  string    `gorm:"size:50" json:"last_name"`
	City      string    `gorm:"size:50" json:"city"`
	State     string    `gorm:"size:50" json:"state"`
	Country   string    `gorm:"size:50" json:"country"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name for DMRUser
func (DMRUser) TableName() string {
	return "dmr_users"
}

// FullName returns the full name of the user
func (u *DMRUser) FullName() string {
	if u.FirstName != "" && u.LastName != "" {
		return u.FirstName + " " + u.LastName
	}
	if u.FirstName != "" {
		return u.FirstName
	}
	if u.LastName != "" {
		return u.LastName
	}
	return ""
}

// Location returns the formatted location string
func (u *DMRUser) Location() string {
	parts := make([]string, 0, 3)
	if u.City != "" {
		parts = append(parts, u.City)
	}
	if u.State != "" {
		parts = append(parts, u.State)
	}
	if u.Country != "" {
		parts = append(parts, u.Country)
	}
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += ", "
		}
		result += part
	}
	return result
}
