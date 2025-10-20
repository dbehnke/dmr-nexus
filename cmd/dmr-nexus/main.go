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
	"github.com/dbehnke/dmr-nexus/pkg/metrics"
	"github.com/dbehnke/dmr-nexus/pkg/mqtt"
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

	// Initialize wait group for goroutines
	var wg sync.WaitGroup

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector()

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

	// Start web server if enabled
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

	// Stop MQTT publisher if running
	if mqttPublisher != nil {
		mqttPublisher.Stop()
	}

	// Wait for all components to stop
	wg.Wait()

	log.Info("DMR-Nexus stopped")
}
