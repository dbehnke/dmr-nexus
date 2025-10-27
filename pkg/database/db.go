package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	// Use modernc.org/sqlite (pure Go, no CGO)
	"gorm.io/driver/sqlite"
	_ "modernc.org/sqlite"
)

// DB wraps the GORM database connection
type DB struct {
	db     *gorm.DB
	logger *logger.Logger
}

// Config holds database configuration
type Config struct {
	Path string // Path to SQLite database file
}

// NewDB creates a new database connection
func NewDB(cfg Config, log *logger.Logger) (*DB, error) {
	if cfg.Path == "" {
		cfg.Path = "dmr-nexus.db"
	}

	// Ensure directory exists
	dir := filepath.Dir(cfg.Path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Configure GORM logger to use our logger
	gormLog := gormlogger.New(
		&gormLogAdapter{log: log},
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// Open database with modernc.org/sqlite (pure Go) driver
	// Using the Dialector interface to specify the pure Go driver
	dialector := sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        cfg.Path,
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormLog,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}
	
	// Set WAL mode and optimize for concurrent access
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	if _, err := sqlDB.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		return nil, fmt.Errorf("failed to set synchronous mode: %w", err)
	}
	if _, err := sqlDB.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Run migrations
	if err := db.AutoMigrate(&Transmission{}, &DMRUser{}); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Info("Database initialized", logger.String("path", cfg.Path))

	return &DB{
		db:     db,
		logger: log,
	}, nil
}

// Close closes the database connection
func (d *DB) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB returns the underlying GORM database instance
func (d *DB) GetDB() *gorm.DB {
	return d.db
}

// gormLogAdapter adapts our logger to GORM's logger interface
type gormLogAdapter struct {
	log *logger.Logger
}

func (l *gormLogAdapter) Printf(format string, args ...interface{}) {
	l.log.Info(fmt.Sprintf(format, args...))
}
