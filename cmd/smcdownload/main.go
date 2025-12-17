package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
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
	var outputPath string

	flag.StringVar(&configPath, "config", DefaultConfigPath, "Path to configuration file")
	flag.StringVar(&dotEnvPath, "dotenv", "", "Path to .env file (overrides config file setting)")
	flag.StringVar(&outputPath, "output", "", "Path to output JSON file")
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
		os.Exit(1)
	}

	if err := smcProvider.Ping(context.Background()); err != nil {
		logger.Error("Failed to ping SmartCitizen API", "error", err)
		os.Exit(1)
	}

	user, err := smcProvider.GetMe(context.Background())
	if err != nil {
		logger.Error("Failed to get authenticated user", "error", err)
		os.Exit(1)
	}

	result := Result{
		User:    user,
		Devices: make([]smartcitizen.DeviceDetail, 0),
	}

	for _, device := range user.Devices {
		logger.Info("User device", "deviceID", device.ID, "name", device.Name, "state", device.State)
		deviceDetail, err := smcProvider.GetDevice(context.Background(), device.ID)
		if err != nil {
			logger.Error("Failed to get device detail", "deviceID", device.ID, "error", err)
			os.Exit(1)
		}

		if deviceDetail == nil {
			logger.Warn("Device detail is nil", "deviceID", device.ID)
			continue
		}

		logger.Info("Fetched device detail", "deviceID", deviceDetail.ID, "name", deviceDetail.Name, "state", deviceDetail.State, "sensorsCount", len(deviceDetail.Data.Sensors))
		result.Devices = append(result.Devices, *deviceDetail)
	}

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal result to JSON", "error", err)
		os.Exit(1)
	}

	buf := bytes.NewReader(jsonResult)
	if outputPath != "" {
		if err := saveFile(outputPath, buf); err != nil {
			logger.Error("Failed to save result to file", "error", err, "path", outputPath)
			os.Exit(1)
		}
		logger.Info("Result saved to JSON file", "path", outputPath)
	} else {
		fmt.Println(string(jsonResult))
	}
}

func initSmartCitizenProvider(appConfig AppConfig, logger *slog.Logger) (*smartcitizen.HTTPProvider, error) {
	smcCredProvider := smartcitizen.NewUserCredentialEnvProvider(appConfig.Smc.UsernameEnv, appConfig.Smc.PasswordEnv, appConfig.Smc.TokenEnv)
	credentials, err := smcCredProvider.Retrieve(context.Background())
	if err != nil {
		logger.Error("Failed to retrieve SmartCitizen credentials", "error", err)
		return nil, fmt.Errorf("failed to retrieve SmartCitizen credentials: %w", err)
	}

	smcProvider := smartcitizen.NewHTTPProvider(appConfig.Smc,
		httpclient.NewDefaultHTTPClient(),
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
