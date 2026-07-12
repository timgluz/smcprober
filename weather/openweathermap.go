package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/timgluz/smcprober/httpclient"
	"github.com/timgluz/smcprober/metric"
)

type HTTPProvider struct {
	config        Config
	apiKey        string
	client        *http.Client // OpenWeatherMap (current weather)
	climateClient *http.Client // Open-Meteo (climate normals) — separate client/timeout/histogram: different vendor, much larger payload
	registry      metric.Registry
	logger        *slog.Logger
}

func NewHTTPProvider(config Config, apiKey string, client *http.Client, registry metric.Registry, logger *slog.Logger) *HTTPProvider {
	// Ensure non-nil transport (mirror smartcitizen pattern)
	if client == nil {
		client = &http.Client{}
	}
	if client.Transport == nil {
		client.Transport = http.DefaultTransport
	}

	histogram := registry.GetOrCreateHistogramVec(
		"api_request_duration_seconds",
		"Duration of HTTP requests to OpenWeatherMap API",
		[]float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0},
		[]string{"endpoint", "status", "method"},
	)

	if transport, ok := client.Transport.(*http.Transport); ok {
		client.Transport = httpclient.NewInstrumentedTransport(transport, histogram)
	} else {
		logger.Warn("HTTP transport is not *http.Transport; metrics instrumentation not applied",
			"transport_type", fmt.Sprintf("%T", client.Transport))
	}

	climateHistogram := registry.GetOrCreateHistogramVec(
		"climate_api_request_duration_seconds",
		"Duration of HTTP requests to Open-Meteo Climate API",
		[]float64{0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0},
		[]string{"endpoint", "status", "method"},
	)
	climateClient := httpclient.NewDefaultHTTPClient()
	climateClient.Timeout = 60 * time.Second // archive queries fetch far more data than a current-weather call
	if transport, ok := climateClient.Transport.(*http.Transport); ok {
		climateClient.Transport = httpclient.NewInstrumentedTransport(transport, climateHistogram)
	} else {
		logger.Warn("Climate HTTP transport is not *http.Transport; metrics instrumentation not applied",
			"transport_type", fmt.Sprintf("%T", climateClient.Transport))
	}

	return &HTTPProvider{
		config:        config,
		apiKey:        apiKey,
		client:        client,
		climateClient: climateClient,
		registry:      registry,
		logger:        logger,
	}
}

func (p *HTTPProvider) Ping(ctx context.Context) error {
	if len(p.config.Locations) == 0 {
		return fmt.Errorf("weather: no locations configured")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	loc := p.config.Locations[0]
	_, err := p.GetCurrentWeather(ctx, loc.Lat, loc.Lon)
	return err
}

// owmCurrentData matches one element of the "data" array returned by
// the /data/4.0/onecall/current endpoint (timemachine-style response).
type owmCurrentData struct {
	Temp       float64 `json:"temp"`
	FeelsLike  float64 `json:"feels_like"`
	Pressure   float64 `json:"pressure"`
	Humidity   float64 `json:"humidity"`
	UVI        float64 `json:"uvi"`
	Clouds     float64 `json:"clouds"`
	Visibility float64 `json:"visibility"`
	WindSpeed  float64 `json:"wind_speed"`
	WindDeg    float64 `json:"wind_deg"`
	Sunrise    int64   `json:"sunrise"`
	Sunset     int64   `json:"sunset"`
}

type owmResponse struct {
	Lat  float64          `json:"lat"`
	Lon  float64          `json:"lon"`
	Data []owmCurrentData `json:"data"`
}

func (p *HTTPProvider) GetCurrentWeather(ctx context.Context, lat, lon float64) (CurrentWeather, error) {
	endpoint, err := url.JoinPath(p.config.Endpoint, p.config.APIPath, "onecall/current")
	if err != nil {
		return CurrentWeather{}, fmt.Errorf("weather: failed to build URL: %w", err)
	}

	params := url.Values{}
	params.Set("lat", strconv.FormatFloat(lat, 'f', -1, 64))
	params.Set("lon", strconv.FormatFloat(lon, 'f', -1, 64))
	params.Set("units", "metric")
	params.Set("appid", p.apiKey)

	fullURL := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return CurrentWeather{}, fmt.Errorf("weather: failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return CurrentWeather{}, fmt.Errorf("weather: request failed: %w", err)
	}

	defer func() {
		// Drain the response body to allow connection reuse
		_, _ = io.Copy(io.Discard, resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			p.logger.Warn("Failed to close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return CurrentWeather{}, fmt.Errorf("weather: unexpected status code: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return CurrentWeather{}, fmt.Errorf("weather: failed to read response body: %w", err)
	}

	var owm owmResponse
	if err := json.Unmarshal(content, &owm); err != nil {
		return CurrentWeather{}, fmt.Errorf("weather: failed to parse response: %w", err)
	}
	if len(owm.Data) == 0 {
		return CurrentWeather{}, fmt.Errorf("weather: empty data array in response")
	}
	d := owm.Data[0]

	cw := CurrentWeather{
		Lat:         owm.Lat,
		Lon:         owm.Lon,
		Temperature: d.Temp,
		FeelsLike:   d.FeelsLike,
		Pressure:    d.Pressure,
		Humidity:    d.Humidity,
		UVIndex:     d.UVI,
		Clouds:      d.Clouds,
		Visibility:  d.Visibility,
		WindSpeed:   d.WindSpeed,
		WindDeg:     d.WindDeg,
		Sunrise:     d.Sunrise,
		Sunset:      d.Sunset,
		// Name and Country are intentionally left empty:
		// Name is injected by the exporter from config.Locations
		// Country is not returned by the onecall endpoint
	}

	return cw, nil
}
