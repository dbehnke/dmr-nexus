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

	"github.com/dbehnke/dmr-nexus/pkg/bridge"
	"github.com/dbehnke/dmr-nexus/pkg/config"
	"github.com/dbehnke/dmr-nexus/pkg/database"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/dbehnke/dmr-nexus/pkg/metrics"
	"github.com/dbehnke/dmr-nexus/pkg/mqtt"
	"github.com/dbehnke/dmr-nexus/pkg/network"
	"github.com/dbehnke/dmr-nexus/pkg/peer"
	"github.com/dbehnke/dmr-nexus/pkg/web"
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
		fmt.Printf("DMR-Nexus %s\n", version)
		fmt.Printf("Git Commit: %s\n", gitCommit)
		fmt.Printf("Built: %s\n", buildTime)
		os.Exit(0)
	}

	// Initialize logger (basic console logger for startup messages)
	log := logger.New(logger.Config{
		Level:  "info",
		Format: "text",
	})

	log.Info("Starting DMR-Nexus",
		logger.String("version", version),
		logger.String("commit", gitCommit),
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

	// Reinitialize logger with config from file
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

	// Initialize wait group for goroutines
	var wg sync.WaitGroup

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector()

	// Initialize database
	db, err := database.NewDB(database.Config{
		Path: "data/dmr-nexus.db",
	}, log.WithComponent("database"))
	if err != nil {
		log.Error("Failed to initialize database", logger.Error(err))
		os.Exit(1)
	}
	defer db.Close()

	txRepo := database.NewTransmissionRepository(db.GetDB())
	log.Info("Database initialized")

	// Start Prometheus metrics server if enabled
	if cfg.Metrics.Enabled && cfg.Metrics.Prometheus.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			metricsServer := metrics.NewPrometheusServer(
				metrics.PrometheusConfig{
					Enabled: cfg.Metrics.Prometheus.Enabled,
					Port:    cfg.Metrics.Prometheus.Port,
					Path:    cfg.Metrics.Prometheus.Path,
				},
				metricsCollector,
				log.WithComponent("metrics"),
			)
			if err := metricsServer.Start(ctx); err != nil && err != context.Canceled {
				log.Error("Prometheus metrics server error", logger.Error(err))
			}
		}()
		log.Info("Prometheus metrics server started",
			logger.Int("port", cfg.Metrics.Prometheus.Port),
			logger.String("path", cfg.Metrics.Prometheus.Path))
	}

	// Initialize MQTT publisher if enabled
	var mqttPublisher *mqtt.Publisher
	if cfg.MQTT.Enabled {
		mqttPublisher = mqtt.New(
			mqtt.Config{
				Enabled:     cfg.MQTT.Enabled,
				Broker:      cfg.MQTT.Broker,
				TopicPrefix: cfg.MQTT.TopicPrefix,
				ClientID:    cfg.MQTT.ClientID,
				Username:    cfg.MQTT.Username,
				Password:    cfg.MQTT.Password,
				QoS:         cfg.MQTT.QoS,
				Retained:    cfg.MQTT.Retained,
			},
			log.WithComponent("mqtt"),
		)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := mqttPublisher.Start(ctx); err != nil && err != context.Canceled {
				log.Error("MQTT publisher error", logger.Error(err))
			}
		}()
		log.Info("MQTT publisher started",
			logger.String("broker", cfg.MQTT.Broker),
			logger.String("topic_prefix", cfg.MQTT.TopicPrefix))
	}

	// Initialize DMR components
	peerManager := peer.NewPeerManager()
	router := bridge.NewRouter()

	// Set up transmission logger for router
	txLogger := bridge.NewTransmissionLogger(txRepo, log.WithComponent("txlog"))
	router.SetTransmissionLogger(txLogger)

	// Start cleanup routine for stale streams
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				txLogger.CleanupStaleStreams(60 * time.Second)
			}
		}
	}()

	// Start web server if enabled (after creating peer manager and router)
	var webServer *web.Server
	if cfg.Web.Enabled {
		webServer = web.NewServer(cfg.Web, log.WithComponent("web")).
			WithPeerManager(peerManager).
			WithRouter(router)

		// Set transmission repository for API
		webServer.GetAPI().SetTransmissionRepo(txRepo)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := webServer.Start(ctx); err != nil && err != context.Canceled {
				log.Error("Web server error", logger.Error(err))
			}
		}()
		log.Info("Web server started",
			logger.String("host", cfg.Web.Host),
			logger.Int("port", cfg.Web.Port))
	}

	// Start DMR network servers for each configured system
	for name, system := range cfg.Systems {
		if !system.Enabled {
			log.Info("System disabled, skipping",
				logger.String("system", name))
			continue
		}

		switch system.Mode {
		case "MASTER":
			log.Info("Starting MASTER mode server",
				logger.String("system", name),
				logger.Int("port", system.Port))

			server := network.NewServer(system, name, log.WithComponent("network."+name)).
				WithPeerManager(peerManager).
				WithRouter(router)

			// Wire peer event handlers to WebSocket if web server is enabled
			if webServer != nil {
				server.SetPeerEventHandlers(
					webServer.PeerConnectedHandler(),
					webServer.PeerDisconnectedHandler(),
				)
			}

			wg.Add(1)
			go func(sysName string, srv *network.Server) {
				defer wg.Done()
				if err := srv.Start(ctx); err != nil && err != context.Canceled {
					log.Error("DMR server error",
						logger.String("system", sysName),
						logger.Error(err))
				}
			}(name, server)

		case "PEER":
			log.Info("PEER mode not yet implemented",
				logger.String("system", name))
			// TODO: Implement PEER mode client

		case "OPENBRIDGE":
			log.Info("OPENBRIDGE mode not yet implemented",
				logger.String("system", name))
			// TODO: Implement OPENBRIDGE mode

		default:
			log.Warn("Unknown system mode",
				logger.String("system", name),
				logger.String("mode", system.Mode))
		}
	}

	log.Info("DMR-Nexus initialized",
		logger.String("server_name", cfg.Server.Name))

	// Wait for shutdown signal
	sig := <-sigChan
	log.Info("Received shutdown signal",
		logger.String("signal", sig.String()))

	// Cancel context to trigger graceful shutdown
	cancel()

	// Stop MQTT publisher if running
	if mqttPublisher != nil {
		mqttPublisher.Stop()
	}

	// Wait for all components to stop
	wg.Wait()

	log.Info("DMR-Nexus stopped")
}
