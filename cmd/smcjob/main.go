package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"github.com/timgluz/smcprober/alert"
	"github.com/timgluz/smcprober/httpclient"
	"github.com/timgluz/smcprober/metric"
	"github.com/timgluz/smcprober/ntfy"
	"github.com/timgluz/smcprober/smartcitizen"
)

const (
	DefaultConfigPath        = "configs/config.json"
	DefaultBatterySensorName = "Battery SCK"

	DeviceStateMetricName = "Device State"
)

type AppConfig struct {
	BatterySensorName string `json:"battery_sensor_name"`

	LogLevel   string `json:"log_level"`
	DotEnvPath string `json:"dotenv_path"`

	Ntfy ntfy.Config         `json:"ntfy"`
	Smc  smartcitizen.Config `json:"smartcitizen"`
}

func main() {
	var configPath string
	var dotEnvPath string

	flag.StringVar(&configPath, "config", DefaultConfigPath, "Path to configuration file")
	flag.StringVar(&dotEnvPath, "dotenv", "", "Path to .env file (overrides config file setting)")
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

	// Create shared metric registry
	namespace := "smartcitizen"
	registry := metric.NewNamespacedRegistry(namespace, logger)

	smcProvider, err := initSmartCitizenProvider(appConfig, registry, logger)
	if err != nil {
		logger.Error("Failed to initialize SmartCitizen provider", "error", err)
		panic(err)
	}

	if err := smcProvider.Ping(context.Background()); err != nil {
		logger.Error("Failed to ping SmartCitizen API", "error", err)
		os.Exit(1)
	}

	user, err := smcProvider.GetMe(context.Background())
	if err != nil {
		logger.Error("Failed to get authenticated user", "error", err)
		panic(err)
	}

	logger.Info("Authenticated user", "userID", user.ID, "username", user.Username)
	notifier, err := initNtfyNotifier(appConfig, logger)
	if err != nil {
		logger.Error("Failed to initialize ntfy notifier", "error", err)
		panic(err)
	}

	alertEngine, err := initAlertEngine(appConfig, notifier, logger)
	if err != nil {
		logger.Error("Failed to initialize alert engine", "error", err)
		panic(err)
	}

	for _, device := range user.Devices {
		logger.Info("User device", "deviceID", device.ID, "name", device.Name, "state", device.State)
		deviceDetail, err := smcProvider.GetDevice(context.Background(), device.ID)
		if err != nil {
			panic(err)
		}

		if deviceDetail == nil {
			logger.Warn("Device detail is nil", "deviceID", device.ID)
			continue
		}

		logger.Info("Fetched device detail", "deviceID", deviceDetail.ID, "name", deviceDetail.Name, "state", deviceDetail.State, "sensorsCount", len(deviceDetail.Data.Sensors))

		evaluateDevice(alertEngine, deviceDetail)
	}
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

	config.Ntfy.ApplyDefaults()
	config.Smc.ApplyDefaults()

	return config, nil
}

func initNtfyNotifier(appConfig AppConfig, logger *slog.Logger) (*ntfy.HTTPNotifier, error) {
	notifier := ntfy.NewHTTPNotifier(appConfig.Ntfy.Endpoint, httpclient.NewDefaultHTTPClient(), logger)

	if appConfig.Ntfy.TokenEnv != "" {
		ntfyCredProvider := ntfy.NewTokenCredentialEnvProvider(appConfig.Ntfy.TokenEnv)
		if err := notifier.SetCredentialProvider(ntfyCredProvider); err != nil {
			logger.Warn("Failed to set ntfy credential provider", "error", err)
		}
	}

	return notifier, nil
}

func initSmartCitizenProvider(appConfig AppConfig, registry metric.Registry, logger *slog.Logger) (*smartcitizen.HTTPProvider, error) {
	smcCredProvider := smartcitizen.NewUserCredentialEnvProvider(appConfig.Smc.UsernameEnv, appConfig.Smc.PasswordEnv, appConfig.Smc.TokenEnv)
	credentials, err := smcCredProvider.Retrieve(context.Background())
	if err != nil {
		logger.Error("Failed to retrieve SmartCitizen credentials", "error", err)
		panic(err)
	}

	smcProvider := smartcitizen.NewHTTPProvider(appConfig.Smc,
		httpclient.NewDefaultHTTPClient(),
		registry,
		logger,
	)

	if err := smcProvider.Authenticate(context.Background(), credentials); err != nil {
		logger.Error("Failed to authenticate with SmartCitizen API", "error", err)
		panic(err)
	}

	return smcProvider, nil
}

func initAlertEngine(appConfig AppConfig, notifier ntfy.Notifier, logger *slog.Logger) (*alert.AlertingEngine, error) {
	engine := alert.NewAlertingEngine(logger)

	batterySensorName := appConfig.BatterySensorName

	engine.AddRule(alert.AlertRule{
		ID:         "battery_ok",
		Name:       "Battery Level OK",
		MetricName: batterySensorName,
		Enabled:    true,
		Condition: func(metric alert.Metric) bool {
			return metric.Name == batterySensorName && metric.Value >= 15.0
		},
		Action: alert.LogAction(logger),
	})

	engine.AddRule(alert.AlertRule{
		ID:         "battery_low",
		Name:       "Battery Level Low",
		MetricName: batterySensorName,
		Enabled:    true,
		Condition: func(metric alert.Metric) bool {
			return metric.Name == batterySensorName && metric.Value < 15.0 && metric.Value >= 10.0
		},
		Action: alert.MultiAction(
			alert.LogAction(logger),
			SendNotificationAction(notifier, appConfig.Ntfy.Topic, "Battery level is low"),
		),
	})

	engine.AddRule(alert.AlertRule{
		ID:         "battery_critical_low",
		Name:       "Battery Level Low",
		MetricName: batterySensorName,
		Enabled:    true,
		Condition: func(metric alert.Metric) bool {
			return metric.Name == batterySensorName && metric.Value < 10.0
		},
		Action: alert.MultiAction(
			alert.LogAction(logger),
			SendNotificationAction(notifier, appConfig.Ntfy.Topic, "Battery level is critically low"),
		),
	})

	engine.AddRule(alert.AlertRule{
		ID:         "device_online",
		Name:       "Device Online",
		MetricName: DeviceStateMetricName,
		Enabled:    true,
		Condition:  alert.ThresholdEquals(smartcitizen.DeviceStateOnline),
		Action:     alert.LogAction(logger),
	})

	engine.AddRule(alert.AlertRule{
		ID:         "device_offline",
		Name:       "Device Offline",
		MetricName: DeviceStateMetricName,
		Enabled:    true,
		Condition:  alert.ThresholdEquals(smartcitizen.DeviceStateOffline),
		Action: alert.MultiAction(
			alert.LogAction(logger),
			SendNotificationAction(notifier, appConfig.Ntfy.Topic, "Device is offline"),
		),
	})

	return engine, nil
}

func SendNotificationAction(notifier ntfy.Notifier, topic string, message string) alert.RuleAction {
	return func(metric alert.Metric, rule alert.AlertRule) error {
		notification := ntfy.Notification{
			Topic:   topic,
			Title:   "Alert: " + rule.Name,
			Message: message,
		}

		return notifier.Send(context.Background(), notification)
	}
}

func evaluateDevice(engine *alert.AlertingEngine, deviceDetail *smartcitizen.DeviceDetail) {
	metrics := mapDeviceSensorsToMetrics(deviceDetail.Data.Sensors)
	// add device-level metrics if needed
	stateMetric := mapDeviceStateToMetric(deviceDetail)
	metrics = append(metrics, stateMetric)

	for _, metric := range metrics {
		engine.Evaluate(metric)
	}
}

func mapDeviceStateToMetric(deviceDetail *smartcitizen.DeviceDetail) alert.Metric {
	return alert.Metric{
		Name:        DeviceStateMetricName,
		Description: fmt.Sprintf("Device state: %s", deviceDetail.State),
		Value:       deviceDetail.StateValue(),
		Unit:        "state",
		Timestamp:   smartcitizen.ParseTimeToUnix(deviceDetail.UpdatedAt),
	}
}

func mapDeviceSensorsToMetrics(sensors []smartcitizen.DeviceSensor) []alert.Metric {
	metrics := make([]alert.Metric, 0, len(sensors))
	for _, sensor := range sensors {
		metrics = append(metrics, mapDeviceSensorToMetric(sensor))
	}
	return metrics
}

func mapDeviceSensorToMetric(sensor smartcitizen.DeviceSensor) alert.Metric {
	return alert.Metric{
		Name:        sensor.Name,
		Description: sensor.Description,
		Value:       sensor.Value,
		Unit:        sensor.Unit,
		Timestamp:   sensor.ToUnix(),
	}
}
