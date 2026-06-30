package weather

import "context"

const CurrentWeatherType = "CurrentWeather" // used by CombinedConverter reflect dispatch

type CurrentWeather struct {
	Name        string  // injected by exporter from config, not by provider
	Lat, Lon    float64
	Country     string  // always "" in v1 — onecall endpoint does not return country
	Temperature float64
	FeelsLike   float64
	Humidity    float64
	Pressure    float64
	WindSpeed   float64
	WindDeg     float64
	Clouds      float64
	Visibility  float64
	UVIndex     float64
	Sunrise     int64
	Sunset      int64
}

type Provider interface {
	Ping(ctx context.Context) error
	GetCurrentWeather(ctx context.Context, lat, lon float64) (CurrentWeather, error)
}
