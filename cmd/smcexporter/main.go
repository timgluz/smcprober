package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/timgluz/smcprober/httpclient"
	"github.com/timgluz/smcprober/smartcitizen"
)

const (
	DefaultConfigPath        = "configs/config.json"
	DefaultBatterySensorName = "Battery SCK"

	DeviceStateMetricName = "Device State"
)

type AppConfig struct {
	LogLevel   string `json:"log_level"`
	DotEnvPath string `json:"dotenv_path"`

	Smc smartcitizen.Config `json:"smartcitizen"`
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
		Level: slog.LevelInfo,
	}))

	smcProvider, err := initSmartCitizenProvider(appConfig, logger)
	if err != nil {
		logger.Error("Failed to initialize SmartCitizen provider", "error", err)
		panic(err)
	}

	if err := smcProvider.Ping(context.Background()); err != nil {
		logger.Error("Failed to ping SmartCitizen API", "error", err)
		os.Exit(1)
	}

	exporter := smartcitizen.NewAPIExporter(appConfig.Smc, smcProvider, logger)

	// Start background updater
	go exporter.Start(15 * time.Second)

	// HTTP handlers
	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>SmartCitizen Exporter</title></head>
			<body>
			<h1>Prometheus Exporter for SmartCitizen devices</h1>
			<p><a href="/metrics">Metrics</a></p>
			<p>Metrics are dynamically registered and updated</p>
			</body>
			</html>`))
	})

	logger.Info("Starting exporter", "port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Error("Error starting server", "error", err)
	}

}

func initSmartCitizenProvider(appConfig AppConfig, logger *slog.Logger) (*smartcitizen.HTTPProvider, error) {
	smcCredProvider := smartcitizen.NewUserCredentialEnvProvider(appConfig.Smc.UsernameEnv, appConfig.Smc.PasswordEnv, appConfig.Smc.TokenEnv)
	credentials, err := smcCredProvider.Retrieve(context.Background())
	if err != nil {
		logger.Error("Failed to retrieve SmartCitizen credentials", "error", err)
		panic(err)
	}

	smcProvider := smartcitizen.NewHTTPProvider(appConfig.Smc,
		httpclient.NewDefaultHTTPClient(),
		logger,
	)

	if err := smcProvider.Authenticate(context.Background(), credentials); err != nil {
		logger.Error("Failed to authenticate with SmartCitizen API", "error", err)
		panic(err)
	}

	return smcProvider, nil
}

func loadConfigFromJSONFile(path string) (AppConfig, error) {
	var config AppConfig
	file, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return config, err
	}

	config.Smc.ApplyDefaults()

	return config, nil
}

func saveFile(path string, reader io.Reader) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return err
	}

	return nil
}
