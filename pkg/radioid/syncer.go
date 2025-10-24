package radioid

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/database"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

const (
	// RadioIDURL is the URL to download the DMR user database
	RadioIDURL = "https://radioid.net/static/user.csv"
	// SyncInterval is how often to sync the database (24 hours)
	SyncInterval = 24 * time.Hour
	// BatchSize for database upserts
	BatchSize = 1000
)

// Syncer handles syncing the RadioID database
type Syncer struct {
	repo   *database.DMRUserRepository
	logger *logger.Logger
	client *http.Client
}

// NewSyncer creates a new RadioID syncer
func NewSyncer(repo *database.DMRUserRepository, log *logger.Logger) *Syncer {
	return &Syncer{
		repo:   repo,
		logger: log,
		client: &http.Client{
			Timeout: 5 * time.Minute, // Large file, need generous timeout
		},
	}
}

// Start begins the periodic sync process
func (s *Syncer) Start(ctx context.Context) {
	// Sync immediately on startup
	s.logger.Info("Starting RadioID database sync")
	if err := s.Sync(ctx); err != nil {
		s.logger.Error("Failed to sync RadioID database on startup", logger.Error(err))
	}

	// Set up periodic sync
	ticker := time.NewTicker(SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("RadioID syncer stopped")
			return
		case <-ticker.C:
			s.logger.Info("Starting periodic RadioID database sync")
			if err := s.Sync(ctx); err != nil {
				s.logger.Error("Failed to sync RadioID database", logger.Error(err))
			}
		}
	}
}

// Sync downloads and parses the RadioID database
func (s *Syncer) Sync(ctx context.Context) error {
	start := time.Now()
	s.logger.Info("Downloading RadioID database", logger.String("url", RadioIDURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, RadioIDURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download database: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			s.logger.Warn("Failed to close response body", logger.Error(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse CSV
	users, err := s.parseCSV(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse CSV: %w", err)
	}

	s.logger.Info("Parsed RadioID database",
		logger.Int("users", len(users)))

	// Upsert users in batches
	if err := s.repo.UpsertBatch(users, BatchSize); err != nil {
		return fmt.Errorf("failed to save users: %w", err)
	}

	// Get final count
	count, _ := s.repo.Count()

	duration := time.Since(start)
	s.logger.Info("RadioID database sync complete",
		logger.Int64("total_users", count),
		logger.String("duration", duration.String()))

	return nil
}

// parseCSV parses the RadioID CSV format
// Expected format: RADIO_ID,CALLSIGN,FIRST_NAME,LAST_NAME,CITY,STATE,COUNTRY,...
func (s *Syncer) parseCSV(r io.Reader) ([]database.DMRUser, error) {
	reader := csv.NewReader(bufio.NewReader(r))

	// Skip header row
	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	users := make([]database.DMRUser, 0, 100000) // Pre-allocate for typical size
	lineNum := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.logger.Warn("Error reading CSV line",
				logger.Int("line", lineNum),
				logger.Error(err))
			lineNum++
			continue
		}
		lineNum++

		// Need at least 7 columns: RADIO_ID,CALLSIGN,FIRST_NAME,LAST_NAME,CITY,STATE,COUNTRY
		if len(record) < 7 {
			continue
		}

		// Parse radio ID
		radioID, err := strconv.ParseUint(record[0], 10, 32)
		if err != nil {
			continue // Skip invalid radio IDs
		}

		user := database.DMRUser{
			RadioID:   uint32(radioID),
			Callsign:  record[1],
			FirstName: record[2],
			LastName:  record[3],
			City:      record[4],
			State:     record[5],
			Country:   record[6],
			UpdatedAt: time.Now(),
		}

		users = append(users, user)
	}

	return users, nil
}
