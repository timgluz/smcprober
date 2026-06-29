package weather

import (
	"context"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/timgluz/smcprober/metric"
)

type WeatherExporter struct {
	config         Config
	provider       Provider
	registry       metric.Registry
	converter      metric.Converter
	logger         *slog.Logger
	requestCounter *prometheus.CounterVec
	errorCounter   *prometheus.CounterVec
}

func NewWeatherExporter(namespace string, config Config, provider Provider, logger *slog.Logger) *WeatherExporter {
	registry := metric.NewNamespacedRegistry(namespace, logger)
	return NewWeatherExporterWithRegistry(config, provider, registry, logger)
}

func NewWeatherExporterWithRegistry(config Config, provider Provider, registry metric.Registry, logger *slog.Logger) *WeatherExporter {
	converter := metric.NewCombinedConverter()
	converter.Add(NewWeatherInfoConverter(), NewWeatherMetricsConverter())

	requestCounter := registry.GetOrCreateCounterVec(
		"api_requests_total",
		"Total API requests to OpenWeatherMap",
		[]string{locationLabel},
	)
	errorCounter := registry.GetOrCreateCounterVec(
		"api_errors_total",
		"Total API errors from OpenWeatherMap",
		[]string{locationLabel, "type"},
	)

	return &WeatherExporter{
		config:         config,
		provider:       provider,
		registry:       registry,
		converter:      converter,
		logger:         logger,
		requestCounter: requestCounter,
		errorCounter:   errorCounter,
	}
}

func (e *WeatherExporter) updateMetrics(ctx context.Context) {
	for _, loc := range e.config.Locations {
		e.requestCounter.WithLabelValues(loc.Name).Inc()

		cw, err := e.provider.GetCurrentWeather(ctx, loc.Lat, loc.Lon)
		if err != nil {
			e.logger.Error("Failed to get current weather", "location", loc.Name, "error", err)
			e.errorCounter.WithLabelValues(loc.Name, "fetch_error").Inc()
			continue // do NOT abort the loop — other locations should still be processed
		}

		cw.Name = loc.Name // inject location name; provider leaves it empty

		if err := e.converter.Convert(e.registry, cw); err != nil {
			e.logger.Error("Failed to convert weather metrics", "location", loc.Name, "error", err)
			e.errorCounter.WithLabelValues(loc.Name, "convert_error").Inc()
		}
	}
}

func (e *WeatherExporter) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Immediate first call
	e.updateMetrics(ctx)

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("Stopping weather metrics updater", "reason", ctx.Err())
			return
		case <-ticker.C:
			e.updateMetrics(ctx)
		}
	}
}
