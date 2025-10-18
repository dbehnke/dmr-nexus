package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/dbehnke/dmr-nexus/pkg/config"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/dbehnke/dmr-nexus/pkg/web"
)

var (
	version   = "dev"
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
		fmt.Printf("DMR-Nexus %s (built %s)\n", version, buildTime)
		os.Exit(0)
	}

	// Initialize logger (basic console logger for now)
	log := logger.New(logger.Config{
		Level:  "info",
		Format: "text",
	})

	log.Info("Starting DMR-Nexus",
		logger.String("version", version),
		logger.String("build_time", buildTime))

	// Load configuration
	cfg, err := config.Load(*configFile)
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

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start web server if enabled
	var wg sync.WaitGroup
	if cfg.Web.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := web.Start(ctx, cfg.Web, log.WithComponent("web")); err != nil && err != context.Canceled {
				log.Error("Web server error", logger.Error(err))
			}
		}()
		log.Info("Web server started",
			logger.String("host", cfg.Web.Host),
			logger.Int("port", cfg.Web.Port))
	}

	// TODO: Initialize and start the DMR server components
	log.Info("DMR-Nexus initialized",
		logger.String("server_name", cfg.Server.Name))

	// Wait for shutdown signal
	sig := <-sigChan
	log.Info("Received shutdown signal",
		logger.String("signal", sig.String()))

	// Cancel context to trigger graceful shutdown
	cancel()

	// Wait for all components to stop
	wg.Wait()

	log.Info("DMR-Nexus stopped")
}
