package smartcitizen

import (
	"context"
	"log/slog"
	"time"

	"github.com/timgluz/smcprober/metric"
)

// APIExporter uses the metric registry
type APIExporter struct {
	config Config

	provider Provider
	registry *metric.Registry
	logger   *slog.Logger
}

func NewAPIExporter(config Config, provider Provider, logger *slog.Logger) *APIExporter {
	return &APIExporter{
		config:   config,
		provider: provider,
		logger:   logger,
		registry: metric.NewRegistry("smartcitizen"),
	}
}

func (e *APIExporter) fetchAPIData(ctx context.Context) (*UserDeviceCollection, error) {
	defer ctx.Done()

	user, err := e.provider.GetMe(ctx)
	if err != nil {
		e.logger.Error("Failed to get authenticated user", "error", err)
		panic(err)
	}

	result := UserDeviceCollection{
		User:    user,
		Devices: make([]DeviceDetail, 0),
	}

	for _, device := range user.Devices {
		e.logger.Info("User device", "deviceID", device.ID, "name", device.Name, "state", device.State)
		deviceDetail, err := e.provider.GetDevice(ctx, device.ID)
		if err != nil {
			panic(err)
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
	// TODO: Process the data and update metrics accordingly

	for _, device := range data.Devices {
		if err := mapDeviceSensorsToMetrics(e.registry, device.Data.Sensors); err != nil {
			e.logger.Error("Failed to map device sensors to metrics", "error", err, "deviceID", device.ID)
			errCounter := e.registry.GetOrCreateCounterVec(
				"data_errors_total",
				"total data processing errors",
				[]string{"type"},
			)

			errCounter.WithLabelValues("mapping_error").Inc()
			continue
		}
	}
}

func (e *APIExporter) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// TODO: use context with cancellation,
	// Update immediately
	e.updateMetrics(context.Background())

	for range ticker.C {
		e.updateMetrics(context.Background())
	}
}

func mapDeviceSensorsToMetrics(registry *metric.Registry, sensors []DeviceSensor) error {
	//TODO: implement dynamic mapping of sensors to metrics
	return nil
}
