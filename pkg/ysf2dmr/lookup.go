package ysf2dmr

import (
	"fmt"
	"strings"

	"github.com/dbehnke/dmr-nexus/pkg/database"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

// Lookup handles DMR ID lookups from callsigns
type Lookup struct {
	repo   *database.DMRUserRepository
	logger *logger.Logger
}

// NewLookup creates a new DMR ID lookup handler
func NewLookup(repo *database.DMRUserRepository, log *logger.Logger) *Lookup {
	return &Lookup{
		repo:   repo,
		logger: log,
	}
}

// FindID finds a DMR ID from a callsign
// Returns 0 if not found
func (l *Lookup) FindID(callsign string) uint32 {
	// Clean up callsign - remove suffixes and whitespace
	cs := cleanCallsign(callsign)

	if cs == "" {
		return 0
	}

	// Try exact match first
	user, err := l.repo.GetByCallsign(cs)
	if err == nil && user != nil {
		l.logger.Debug("Found DMR ID for callsign",
			logger.String("callsign", cs),
			logger.Uint32("dmr_id", user.RadioID))
		return user.RadioID
	}

	// Try base callsign (without suffix)
	baseCS := getBaseCallsign(cs)
	if baseCS != cs {
		user, err = l.repo.GetByCallsign(baseCS)
		if err == nil && user != nil {
			l.logger.Debug("Found DMR ID for base callsign",
				logger.String("original", cs),
				logger.String("base", baseCS),
				logger.Uint32("dmr_id", user.RadioID))
			return user.RadioID
		}
	}

	l.logger.Debug("No DMR ID found for callsign",
		logger.String("callsign", cs))
	return 0
}

// FindCallsign finds a callsign from a DMR ID
// Returns empty string if not found
func (l *Lookup) FindCallsign(dmrID uint32) string {
	if dmrID == 0 {
		return ""
	}

	user, err := l.repo.GetByRadioID(dmrID)
	if err != nil || user == nil {
		l.logger.Debug("No callsign found for DMR ID",
			logger.Uint32("dmr_id", dmrID))
		return ""
	}

	l.logger.Debug("Found callsign for DMR ID",
		logger.Uint32("dmr_id", dmrID),
		logger.String("callsign", user.Callsign))

	return user.Callsign
}

// FindName finds a full name from a DMR ID
// Returns "Unknown" if not found
func (l *Lookup) FindName(dmrID uint32) string {
	if dmrID == 0 {
		return "Unknown"
	}

	user, err := l.repo.GetByRadioID(dmrID)
	if err != nil || user == nil {
		return "Unknown"
	}

	// Construct full name
	name := strings.TrimSpace(fmt.Sprintf("%s %s", user.FirstName, user.LastName))
	if name == "" {
		return "Unknown"
	}

	return name
}

// Exists checks if a DMR ID exists in the database
func (l *Lookup) Exists(dmrID uint32) bool {
	if dmrID == 0 {
		return false
	}

	user, err := l.repo.GetByRadioID(dmrID)
	return err == nil && user != nil
}

// cleanCallsign removes trailing spaces and converts to uppercase
func cleanCallsign(cs string) string {
	cs = strings.TrimSpace(cs)
	cs = strings.ToUpper(cs)
	return cs
}

// getBaseCallsign extracts the base callsign without suffix
// Examples:
//   - "KB3EFE-N" -> "KB3EFE"
//   - "KB3EFE/M" -> "KB3EFE"
//   - "KB3EFE" -> "KB3EFE"
func getBaseCallsign(cs string) string {
	// Remove suffix after dash or slash
	if idx := strings.IndexAny(cs, "-/"); idx != -1 {
		return cs[:idx]
	}
	return cs
}
