package smartcitizen

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/timgluz/smcprober/metric"
)

// APIExporter uses the metric registry
type APIExporter struct {
	config Config

	provider  Provider
	registry  metric.Registry
	converter metric.Converter
	logger    *slog.Logger

	// Metrics
	dataErrorCounter *prometheus.CounterVec
}

func NewAPIExporter(namespace string, config Config, provider Provider, logger *slog.Logger) *APIExporter {
	registry := metric.NewNamespacedRegistry(namespace, logger)
	sensorMapping := metric.NewSensorMetricMapping()

	return NewAPIExporterWithRegistry(config, provider, registry, sensorMapping, logger)
}

// NewAPIExporterWithRegistry creates a new APIExporter with an existing registry
func NewAPIExporterWithRegistry(config Config, provider Provider,
	registry metric.Registry,
	sensorMapping *metric.SensorMetricMapping,
	logger *slog.Logger,
) *APIExporter {
	// Register converters
	converter := metric.NewCombinedConverter()
	converter.Add(NewDeviceInfoConverter("device_info"),
		NewDeviceStateConverter("device_state"),
		NewDeviceSensorConverter("sensor", sensorMapping),
		NewDeviceSensorInfoConverter("sensor_info"),
	)

	// Create error counter once
	dataErrorCounter := registry.GetOrCreateCounterVec(
		"data_errors_total",
		"Total data processing errors",
		[]string{"type"},
	)

	return &APIExporter{
		config:           config,
		provider:         provider,
		registry:         registry,
		converter:        converter,
		logger:           logger,
		dataErrorCounter: dataErrorCounter,
	}
}

func (e *APIExporter) fetchAPIData(ctx context.Context) (*UserDeviceCollection, error) {
	user, err := e.provider.GetMe(ctx)
	if err != nil {
		e.logger.Error("Failed to get authenticated user", "error", err)
		return nil, fmt.Errorf("failed to get authenticated user: %w", err)
	}

	result := UserDeviceCollection{
		User:    user,
		Devices: make([]DeviceDetail, 0),
	}

	for _, device := range user.Devices {
		e.logger.Info("User device", "deviceID", device.ID, "name", device.Name, "state", device.State)
		deviceDetail, err := e.provider.GetDevice(ctx, device.ID)
		if err != nil {
			e.logger.Error("Failed to get device detail", "deviceID", device.ID, "error", err)
			return nil, fmt.Errorf("failed to get device %d: %w", device.ID, err)
		}

		if deviceDetail == nil {
			e.logger.Warn("Device detail is nil", "deviceID", device.ID)
			continue
		}

		e.logger.Info("Fetched device detail", "deviceID", deviceDetail.ID,
			"name", deviceDetail.Name, "state", deviceDetail.State,
			"sensorsCount", len(deviceDetail.Data.Sensors),
		)
		result.Devices = append(result.Devices, *deviceDetail)
	}

	return &result, nil
}

func (e *APIExporter) updateMetrics(ctx context.Context) {
	e.logger.Info("Updating metrics from SmartCitizen API")
	// Track requests
	reqCounter := e.registry.GetOrCreateCounter(
		"api_requests_total",
		"Total API requests",
	)
	reqCounter.Inc()

	// Fetch data
	data, err := e.fetchAPIData(ctx)
	if err != nil {
		e.logger.Error("Error fetching data", "error", err)
		errCounter := e.registry.GetOrCreateCounterVec(
			"api_errors_total",
			"Total API errors",
			[]string{"type"},
		)
		errCounter.WithLabelValues("fetch_error").Inc()

		return
	}

	successCounter := e.registry.GetOrCreateCounter(
		"api_requests_success_total",
		"Total successful API requests",
	)
	successCounter.Inc()

	// Update metrics dynamically based on API response
	e.processAPIData(data)
}

func (e *APIExporter) processAPIData(data *UserDeviceCollection) {
	if data == nil {
		e.logger.Warn("No data to process")
		return
	}

	// Map user device details to metrics
	for _, device := range data.Devices {
		if err := e.convertDeviceDetailToMetrics(device); err != nil {
			e.logger.Error("Failed to map device detail to metrics", "error", err, "deviceID", device.ID)
			continue
		}

		if err := e.convertDeviceSensorsToMetrics(device.UUID, device.Data.Sensors); err != nil {
			e.logger.Error("Failed to map device sensors to metrics", "error", err, "deviceID", device.ID)
			continue
		}
	}
}

func (e *APIExporter) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if e.registry == nil {
		e.logger.Error("Metric registry is not initialized")
		return
	}

	// Update metrics immediately on start
	e.updateMetrics(ctx)

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("Stopping metrics updater", "reason", ctx.Err())
			return
		case <-ticker.C:
			e.updateMetrics(ctx)
			e.logger.Info("Metrics updated, will update again after interval", "interval", interval)
		}
	}
}

func (e *APIExporter) convertDeviceDetailToMetrics(detail DeviceDetail) error {
	if err := e.converter.Convert(e.registry, detail); err != nil {
		e.logger.Error("Error converting device detail to metrics", "deviceID", detail.ID, "error", err)
		e.dataErrorCounter.WithLabelValues("mapping_error").Inc()
		return err
	}
	return nil
}

func (e *APIExporter) convertDeviceSensorsToMetrics(deviceUUID string, sensors []DeviceSensor) error {
	for _, sensor := range sensors {
		// Ensure sensor has device UUID set
		if sensor.DeviceUUID == "" {
			sensor.DeviceUUID = deviceUUID
		}

		if err := e.converter.Convert(e.registry, sensor); err != nil {
			e.logger.Error("Error converting sensor data to metrics", "sensorID", sensor.ID, "error", err)
			e.dataErrorCounter.WithLabelValues("mapping_error").Inc()
			return err
		}
	}

	return nil
}
