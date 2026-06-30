package weather_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/timgluz/smcprober/metric"
	"github.com/timgluz/smcprober/weather"
)

const owmFixtureJSON = `{
    "lat": 49.4811,
    "lon": 8.4353,
    "data": [
        {
            "temp": 22.1,
            "feels_like": 21.5,
            "pressure": 1013,
            "humidity": 65,
            "uvi": 3.2,
            "clouds": 40,
            "visibility": 10000,
            "wind_speed": 3.5,
            "wind_deg": 180,
            "sunrise": 1719550800,
            "sunset": 1719605400
        }
    ]
}`

func TestHTTPProvider_GetCurrentWeather(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// assert required query params are present
		q := r.URL.Query()
		if q.Get("lat") == "" {
			t.Error("missing lat param")
		}
		if q.Get("lon") == "" {
			t.Error("missing lon param")
		}
		if q.Get("units") != "metric" {
			t.Errorf("units = %q, want metric", q.Get("units"))
		}
		if q.Get("appid") == "" {
			t.Error("missing appid param")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(owmFixtureJSON))
	}))
	defer ts.Close()

	reg := metric.NewNamespacedRegistry(t.Name(), slog.Default())
	cfg := weather.Config{
		Endpoint:  ts.URL,
		APIPath:   "data/4.0",
		TokenEnv:  "X",
		Locations: []weather.Location{{Name: "Test", Lat: 49.4811, Lon: 8.4353}},
	}
	provider := weather.NewHTTPProvider(cfg, "testkey", ts.Client(), reg, slog.Default())

	cw, err := provider.GetCurrentWeather(context.Background(), 49.4811, 8.4353)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cw.Temperature != 22.1 {
		t.Errorf("Temperature = %f, want 22.1", cw.Temperature)
	}
	if cw.UVIndex != 3.2 {
		t.Errorf("UVIndex = %f, want 3.2", cw.UVIndex)
	}
	if cw.Sunrise != 1719550800 {
		t.Errorf("Sunrise = %d, want 1719550800", cw.Sunrise)
	}
	if cw.Humidity != 65.0 {
		t.Errorf("Humidity = %f, want 65.0", cw.Humidity)
	}
}

func TestHTTPProvider_GetCurrentWeather_NonOKStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	reg := metric.NewNamespacedRegistry(t.Name(), slog.Default())
	cfg := weather.Config{
		Endpoint:  ts.URL,
		APIPath:   "data/4.0",
		TokenEnv:  "X",
		Locations: []weather.Location{{Name: "Test", Lat: 49.4811, Lon: 8.4353}},
	}
	provider := weather.NewHTTPProvider(cfg, "badkey", ts.Client(), reg, slog.Default())

	_, err := provider.GetCurrentWeather(context.Background(), 49.4811, 8.4353)
	if err == nil {
		t.Fatal("expected error for non-200 status, got nil")
	}
}
