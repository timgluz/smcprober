package weather

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/timgluz/smcprober/metric"
)

type mockProvider struct {
	weather CurrentWeather
	err     error
	calls   int
}

func (m *mockProvider) Ping(ctx context.Context) error { return nil }
func (m *mockProvider) GetCurrentWeather(ctx context.Context, lat, lon float64) (CurrentWeather, error) {
	m.calls++
	return m.weather, m.err
}

func newTestRegistry(t *testing.T) *metric.NamespacedRegistry {
	t.Helper()
	return metric.NewNamespacedRegistry(t.Name(), slog.Default())
}

const testLocation = "Ludwigshafen"

func TestWeatherExporter_UpdateMetrics_Success(t *testing.T) {
	reg := newTestRegistry(t)
	mock := &mockProvider{
		weather: CurrentWeather{
			Temperature: 22.1,
			UVIndex:     3.2,
			Humidity:    65.0,
		},
	}
	cfg := Config{
		Locations: []Location{{Name: testLocation, Lat: 49.4811, Lon: 8.4353}},
	}
	exp := NewWeatherExporterWithRegistry(cfg, mock, reg, slog.Default())
	exp.updateMetrics(context.Background())

	if mock.calls != 1 {
		t.Errorf("provider called %d times, want 1", mock.calls)
	}

	// temperature gauge must be registered
	if _, ok := reg.GetCollectorByName("temperature"); !ok {
		t.Error("temperature gauge not registered")
	}
	// location_info gauge must be registered
	if _, ok := reg.GetCollectorByName("location_info"); !ok {
		t.Error("location_info gauge not registered")
	}
}

func TestWeatherExporter_UpdateMetrics_ProviderError(t *testing.T) {
	reg := newTestRegistry(t)
	mock := &mockProvider{err: fmt.Errorf("network error")}
	cfg := Config{
		Locations: []Location{{Name: testLocation, Lat: 49.4811, Lon: 8.4353}},
	}
	exp := NewWeatherExporterWithRegistry(cfg, mock, reg, slog.Default())
	exp.updateMetrics(context.Background()) // must not panic

	// error counter must be registered
	if _, ok := reg.GetCollectorByName("api_errors_total"); !ok {
		t.Error("api_errors_total counter not registered")
	}
}

func TestWeatherExporter_UpdateMetrics_MultipleLocations(t *testing.T) {
	reg := newTestRegistry(t)
	cfg := Config{
		Locations: []Location{
			{Name: testLocation, Lat: 49.4811, Lon: 8.4353},
			{Name: "Berlin", Lat: 52.52, Lon: 13.405},
		},
	}

	// Both locations error — verify the loop does not short-circuit (2 calls, no panic)
	errMock := &mockProvider{err: fmt.Errorf("all fail")}
	exp := NewWeatherExporterWithRegistry(cfg, errMock, reg, slog.Default())
	exp.updateMetrics(context.Background())

	if errMock.calls != 2 {
		t.Errorf("provider called %d times with 2 locations, want 2", errMock.calls)
	}
}
