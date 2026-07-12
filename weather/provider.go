package weather

import (
	"context"
	"time"
)

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

const ClimateNormalsType = "ClimateNormals" // used by CombinedConverter reflect dispatch

// ClimateNormals summarizes historical temperature data for one location on
// one day-of-year, computed from a multi-year window of daily observations.
type ClimateNormals struct {
	Name     string // injected by exporter from config, not by provider
	Lat, Lon float64
	Month    time.Month
	Day      int

	TempRecordMin   float64 // lowest daily min recorded in the sample window
	TempRecordMax   float64 // highest daily max recorded in the sample window
	TempAverageMin  float64 // mean of daily minimums across the sample window
	TempAverageMax  float64 // mean of daily maximums across the sample window
	TempAverageMean float64 // mean of daily means across the sample window
	SampleYears     int     // number of distinct years contributing data
}

type Provider interface {
	Ping(ctx context.Context) error
	GetCurrentWeather(ctx context.Context, lat, lon float64) (CurrentWeather, error)
	GetClimateNormals(ctx context.Context, lat, lon float64, month time.Month, day int) (ClimateNormals, error)
}
