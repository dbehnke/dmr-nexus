package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/database"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/dbehnke/dmr-nexus/pkg/radioid"
	"github.com/dbehnke/dmr-nexus/pkg/ysf2dmr"
)

var (
	version   = "dev"
	gitCommit = "unknown"
	buildTime = "unknown"
)

func main() {
	// Parse command line flags
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	validate := flag.Bool("validate", false, "Validate configuration and exit")
	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("YSF2DMR %s\n", version)
		fmt.Printf("Git Commit: %s\n", gitCommit)
		fmt.Printf("Built: %s\n", buildTime)
		os.Exit(0)
	}

	// Initialize basic logger for startup
	log := logger.New(logger.Config{
		Level:  "info",
		Format: "text",
	})

	log.Info("Starting YSF2DMR Bridge",
		logger.String("version", version),
		logger.String("commit", gitCommit),
		logger.String("build_time", buildTime))

	// Load configuration
	cfg, err := ysf2dmr.Load(*configFile)
	if err != nil {
		log.Error("Failed to load configuration", logger.Error(err))
		os.Exit(1)
	}

	// Validate only mode
	if *validate {
		log.Info("Configuration is valid")
		os.Exit(0)
	}

	log.Info("Configuration loaded successfully",
		logger.String("config_file", *configFile))

	// Reinitialize logger with config settings
	log = logger.New(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})

	log.Debug("Debug logging enabled")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initialize wait group
	var wg sync.WaitGroup

	// Initialize database
	db, err := database.NewDB(database.Config{
		Path: cfg.DMRID.DatabasePath,
	}, log.WithComponent("database"))
	if err != nil {
		log.Error("Failed to initialize database", logger.Error(err))
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("Failed to close database", logger.Error(err))
		}
	}()

	userRepo := database.NewDMRUserRepository(db.GetDB())
	log.Info("Database initialized",
		logger.String("path", cfg.DMRID.DatabasePath))

	// Start RadioID syncer if enabled
	if cfg.DMRID.SyncEnabled {
		radioIDSyncer := radioid.NewSyncer(userRepo, log.WithComponent("radioid"))
		wg.Add(1)
		go func() {
			defer wg.Done()
			radioIDSyncer.Start(ctx)
		}()
		log.Info("RadioID syncer started")
	} else {
		log.Info("RadioID sync disabled")
	}

	// Create lookup handler
	lookup := ysf2dmr.NewLookup(userRepo, log.WithComponent("lookup"))

	// Create and start bridge
	bridge := ysf2dmr.NewBridge(cfg, lookup, log.WithComponent("bridge"))

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bridge.Start(ctx); err != nil {
			log.Error("Bridge error", logger.Error(err))
			cancel()
		}
	}()

	log.Info("YSF2DMR Bridge initialized",
		logger.String("ysf_server", fmt.Sprintf("%s:%d", cfg.YSF.ServerAddress, cfg.YSF.ServerPort)),
		logger.String("dmr_server", fmt.Sprintf("%s:%d", cfg.DMR.ServerAddress, cfg.DMR.ServerPort)),
		logger.Uint32("dmr_id", cfg.DMR.ID),
		logger.Uint32("startup_tg", cfg.DMR.StartupTG))

	// Wait for shutdown signal
	sig := <-sigChan
	log.Info("Received shutdown signal",
		logger.String("signal", sig.String()))

	// Cancel context to trigger graceful shutdown
	cancel()

	// Stop bridge
	if err := bridge.Stop(); err != nil {
		log.Error("Error stopping bridge", logger.Error(err))
	}

	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("Clean shutdown completed")
	case <-time.After(5 * time.Second):
		log.Warn("Shutdown timeout, forcing exit")
	}

	log.Info("YSF2DMR Bridge stopped")
}
