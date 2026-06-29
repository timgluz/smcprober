package weather_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/timgluz/smcprober/metric"
	"github.com/timgluz/smcprober/weather"
)

// newTestRegistry creates a registry with a unique namespace per test to avoid
// duplicate metric registration across tests (NamespacedRegistry registers with
// the global prometheus.DefaultRegisterer). Run tests with -count=1 only.
func newTestRegistry(t *testing.T) *metric.NamespacedRegistry {
	t.Helper()
	return metric.NewNamespacedRegistry(t.Name(), slog.Default())
}

func testWeather() weather.CurrentWeather {
	return weather.CurrentWeather{
		Name:        "Ludwigshafen",
		Lat:         49.4811,
		Lon:         8.4353,
		Country:     "",
		Temperature: 22.1,
		FeelsLike:   21.5,
		Humidity:    65.0,
		Pressure:    1013.0,
		WindSpeed:   3.5,
		WindDeg:     180.0,
		Clouds:      40.0,
		Visibility:  10000.0,
		UVIndex:     3.2,
		Sunrise:     1719550800,
		Sunset:      1719605400,
	}
}

func TestWeatherInfoConverter_Convert(t *testing.T) {
	reg := newTestRegistry(t)
	conv := weather.NewWeatherInfoConverter()
	cw := testWeather()

	if err := conv.Convert(reg, cw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// gauge must exist with correct labels; verify the name is registered
	if _, ok := reg.GetCollectorByName("location_info"); !ok {
		t.Fatal("location_info gauge not registered")
	}
}

func TestWeatherInfoConverter_TypeMismatch(t *testing.T) {
	reg := newTestRegistry(t)
	conv := weather.NewWeatherInfoConverter()

	err := conv.Convert(reg, "not-a-CurrentWeather")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, weather.ErrInvalidDataType) {
		t.Fatalf("expected ErrInvalidDataType, got %v", err)
	}
}

func TestWeatherMetricsConverter_Convert(t *testing.T) {
	reg := newTestRegistry(t)
	conv := weather.NewWeatherMetricsConverter()
	cw := testWeather()

	if err := conv.Convert(reg, cw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All 11 gauges must be registered
	for _, name := range []string{
		"temperature", "feels_like", "humidity", "pressure",
		"wind_speed", "wind_direction", "cloud_coverage", "visibility",
		"uv_index", "sunrise_timestamp", "sunset_timestamp",
	} {
		if _, ok := reg.GetCollectorByName(name); !ok {
			t.Errorf("gauge %q not registered", name)
		}
	}
}

func TestWeatherMetricsConverter_TypeMismatch(t *testing.T) {
	reg := newTestRegistry(t)
	conv := weather.NewWeatherMetricsConverter()

	err := conv.Convert(reg, 42)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, weather.ErrInvalidDataType) {
		t.Fatalf("expected ErrInvalidDataType, got %v", err)
	}
}
