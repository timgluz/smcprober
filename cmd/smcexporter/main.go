package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/timgluz/smcprober/httpclient"
	"github.com/timgluz/smcprober/metric"
	"github.com/timgluz/smcprober/smartcitizen"
)

const DefaultConfigPath = "configs/config.json"

type AppConfig struct {
	ScrapeInterval int    `json:"scrape_interval"`
	LogLevel       string `json:"log_level"`
	DotEnvPath     string `json:"dotenv_path"`

	Smc smartcitizen.Config `json:"smartcitizen"`
}

func (c *AppConfig) ApplyDefaults() {
	if c.ScrapeInterval <= 0 {
		c.ScrapeInterval = 30 // Default to 30 seconds
	}
	c.Smc.ApplyDefaults()
}

func (c *AppConfig) GetScrapeIntervalDuration() time.Duration {
	return time.Duration(c.ScrapeInterval) * time.Second
}

func (c *AppConfig) LogLevelValue() slog.Level {
	switch c.LogLevel {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type Result struct {
	User    smartcitizen.User
	Devices []smartcitizen.DeviceDetail
}

func main() {
	var configPath string
	var dotEnvPath string
	var port string

	flag.StringVar(&configPath, "config", DefaultConfigPath, "Path to configuration file")
	flag.StringVar(&dotEnvPath, "dotenv", "", "Path to .env file (overrides config file setting)")
	flag.StringVar(&port, "port", "8080", "port to run the HTTP server on")
	flag.Parse()

	appConfig, err := loadConfigFromJSONFile(configPath)
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	if dotEnvPath != "" {
		appConfig.DotEnvPath = dotEnvPath
	}

	if appConfig.DotEnvPath != "" {
		fmt.Println("Loading .env file from:", appConfig.DotEnvPath)
		if err := godotenv.Load(appConfig.DotEnvPath); err != nil {
			fmt.Println("Error loading .env file:", err)
			os.Exit(1)
		}
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: appConfig.LogLevelValue(),
	}))

	// Create shared metric registry
	namespace := "smartcitizen"
	registry := metric.NewNamespacedRegistry(namespace, logger)

	smcProvider, err := initSmartCitizenProvider(appConfig, registry, logger)
	if err != nil {
		logger.Error("Failed to initialize SmartCitizen provider", "error", err)
		os.Exit(1)
	}

	if err := smcProvider.Ping(context.Background()); err != nil {
		logger.Error("Failed to ping SmartCitizen API", "error", err)
		os.Exit(1)
	}

	exporter := smartcitizen.NewAPIExporterWithRegistry(appConfig.Smc, smcProvider, registry, logger)

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background updater with cancellable context
	go exporter.Start(ctx, appConfig.GetScrapeIntervalDuration())

	// HTTP handlers
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			logger.Error("Failed to write /health response", "error", err)
			return
		}
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			logger.Error("Failed to write /healthz response", "error", err)
			return
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(`<html>
			<head><title>SmartCitizen Exporter</title></head>
			<body>
			<h1>Prometheus Exporter for SmartCitizen devices</h1>
			<p><a href="/metrics">Metrics</a></p>
			<p>Metrics are dynamically registered and updated</p>
			</body>
			</html>`)); err != nil {
			logger.Error("Failed to write root (/) response", "error", err)
			return
		}
	})

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Channel to listen for errors from the server
	serverErrors := make(chan error, 1)

	// Start HTTP server in a goroutine
	go func() {
		logger.Info("Starting HTTP server", "port", port)
		serverErrors <- server.ListenAndServe()
	}()

	// Channel to listen for interrupt signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or server error
	select {
	case err := <-serverErrors:
		logger.Error("Error starting server", "error", err)
		os.Exit(1)
	case sig := <-shutdown:
		logger.Info("Received shutdown signal", "signal", sig)

		// Cancel the context to stop background updater
		cancel()

		// Give outstanding operations 30 seconds to complete
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		// Gracefully shutdown the HTTP server
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("Error during server shutdown", "error", err)
			if closeErr := server.Close(); closeErr != nil {
				logger.Error("Error closing server", "error", closeErr)
			}
			os.Exit(1)
		}

		logger.Info("Server stopped gracefully")
	}
}

func initSmartCitizenProvider(appConfig AppConfig, registry *metric.NamespacedRegistry, logger *slog.Logger) (*smartcitizen.HTTPProvider, error) {
	smcCredProvider := smartcitizen.NewUserCredentialEnvProvider(appConfig.Smc.UsernameEnv, appConfig.Smc.PasswordEnv, appConfig.Smc.TokenEnv)
	credentials, err := smcCredProvider.Retrieve(context.Background())
	if err != nil {
		logger.Error("Failed to retrieve SmartCitizen credentials", "error", err)
		return nil, fmt.Errorf("failed to retrieve SmartCitizen credentials: %w", err)
	}

	smcProvider := smartcitizen.NewHTTPProvider(appConfig.Smc,
		httpclient.NewDefaultHTTPClient(),
		registry,
		logger,
	)

	if err := smcProvider.Authenticate(context.Background(), credentials); err != nil {
		logger.Error("Failed to authenticate with SmartCitizen API", "error", err)
		return nil, fmt.Errorf("failed to authenticate with SmartCitizen API: %w", err)
	}

	return smcProvider, nil
}

func loadConfigFromJSONFile(path string) (AppConfig, error) {
	var config AppConfig
	file, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to close config file: %v\n", closeErr)
		}
	}()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return config, err
	}

	config.ApplyDefaults()

	return config, nil
}
