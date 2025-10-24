package database

import (
	"gorm.io/gorm"
)

// DMRUserRepository handles DMR user database operations
type DMRUserRepository struct {
	db *gorm.DB
}

// NewDMRUserRepository creates a new DMR user repository
func NewDMRUserRepository(db *gorm.DB) *DMRUserRepository {
	return &DMRUserRepository{db: db}
}

// Upsert creates or updates a DMR user record
func (r *DMRUserRepository) Upsert(user *DMRUser) error {
	// Use GORM's Save which will update if exists (based on primary key) or create if not
	return r.db.Save(user).Error
}

// UpsertBatch efficiently upserts multiple users in a transaction
func (r *DMRUserRepository) UpsertBatch(users []DMRUser, batchSize int) error {
	if len(users) == 0 {
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < len(users); i += batchSize {
			end := i + batchSize
			if end > len(users) {
				end = len(users)
			}
			batch := users[i:end]

			// Use CreateInBatches with OnConflict to handle upserts efficiently
			if err := tx.Save(&batch).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetByRadioID retrieves a user by their radio ID
func (r *DMRUserRepository) GetByRadioID(radioID uint32) (*DMRUser, error) {
	var user DMRUser
	err := r.db.Where("radio_id = ?", radioID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByCallsign retrieves a user by their callsign
func (r *DMRUserRepository) GetByCallsign(callsign string) (*DMRUser, error) {
	var user DMRUser
	err := r.db.Where("callsign = ?", callsign).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Count returns the total number of users in the database
func (r *DMRUserRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&DMRUser{}).Count(&count).Error
	return count, err
}

// DeleteAll removes all users from the database
func (r *DMRUserRepository) DeleteAll() error {
	return r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&DMRUser{}).Error
}
