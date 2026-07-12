package weather

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/timgluz/smcprober/metric"
)

type WeatherExporter struct {
	config                Config
	provider              Provider
	registry              metric.Registry
	converter             metric.Converter
	logger                *slog.Logger
	requestCounter        *prometheus.CounterVec
	errorCounter          *prometheus.CounterVec
	climateRequestCounter *prometheus.CounterVec
	climateErrorCounter   *prometheus.CounterVec
}

func NewWeatherExporter(namespace string, config Config, provider Provider, logger *slog.Logger) *WeatherExporter {
	registry := metric.NewNamespacedRegistry(namespace, logger)
	return NewWeatherExporterWithRegistry(config, provider, registry, logger)
}

func NewWeatherExporterWithRegistry(config Config, provider Provider, registry metric.Registry, logger *slog.Logger) *WeatherExporter {
	converter := metric.NewCombinedConverter()
	converter.Add(NewWeatherInfoConverter(), NewWeatherMetricsConverter(), NewClimateNormConverter())

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
	climateRequestCounter := registry.GetOrCreateCounterVec(
		"climate_normal_requests_total",
		"Total climate normals API requests to Open-Meteo",
		[]string{locationLabel},
	)
	climateErrorCounter := registry.GetOrCreateCounterVec(
		"climate_normal_errors_total",
		"Total climate normals API errors from Open-Meteo",
		[]string{locationLabel, "type"},
	)

	return &WeatherExporter{
		config:                config,
		provider:              provider,
		registry:              registry,
		converter:             converter,
		logger:                logger,
		requestCounter:        requestCounter,
		errorCounter:          errorCounter,
		climateRequestCounter: climateRequestCounter,
		climateErrorCounter:   climateErrorCounter,
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
	e.runPeriodic(ctx, interval, "weather metrics", e.updateMetrics)
}

func (e *WeatherExporter) updateClimateNormals(ctx context.Context) {
	now := time.Now().UTC()
	for _, loc := range e.config.Locations {
		e.climateRequestCounter.WithLabelValues(loc.Name).Inc()

		cn, err := e.provider.GetClimateNormals(ctx, loc.Lat, loc.Lon, now.Month(), now.Day())
		if err != nil {
			if errors.Is(err, ErrNoClimateData) {
				e.logger.Warn("No climate normal data available yet", "location", loc.Name, "error", err)
				e.climateErrorCounter.WithLabelValues(loc.Name, "no_data").Inc()
			} else {
				e.logger.Error("Failed to get climate normals", "location", loc.Name, "error", err)
				e.climateErrorCounter.WithLabelValues(loc.Name, "fetch_error").Inc()
			}
			continue // do NOT abort the loop — other locations should still be processed
		}

		cn.Name = loc.Name // inject location name; provider leaves it empty

		if err := e.converter.Convert(e.registry, cn); err != nil {
			e.logger.Error("Failed to convert climate normal metrics", "location", loc.Name, "error", err)
			e.climateErrorCounter.WithLabelValues(loc.Name, "convert_error").Inc()
		}
	}
}

func (e *WeatherExporter) StartClimateNormals(ctx context.Context, interval time.Duration) {
	e.runPeriodic(ctx, interval, "climate normals", e.updateClimateNormals)
}

// runPeriodic calls update immediately, then again every interval, until ctx
// is cancelled. Shared by Start and StartClimateNormals, which differ only
// in cadence and which update function they drive.
func (e *WeatherExporter) runPeriodic(ctx context.Context, interval time.Duration, name string, update func(context.Context)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Immediate first call
	update(ctx)

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("Stopping updater", "name", name, "reason", ctx.Err())
			return
		case <-ticker.C:
			update(ctx)
		}
	}
}
