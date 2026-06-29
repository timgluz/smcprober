package weather

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/timgluz/smcprober/metric"
)

var ErrInvalidDataType = fmt.Errorf("weather: invalid data type for converter")

// WeatherInfoConverter — static location_info gauge
type WeatherInfoConverter struct{}

func NewWeatherInfoConverter() *WeatherInfoConverter { return &WeatherInfoConverter{} }

func (c *WeatherInfoConverter) Match(name string) bool {
	return name == CurrentWeatherType
}

func (c *WeatherInfoConverter) Convert(registry metric.Registry, data any) error {
	cw, ok := data.(CurrentWeather)
	if !ok {
		return fmt.Errorf("%w: %v", ErrInvalidDataType, reflect.TypeOf(data))
	}

	gauge := registry.GetOrCreateGaugeVec(
		"location_info",
		"Static information about the weather location",
		[]string{"location", "lat", "lon", "country"},
	)
	gauge.With(prometheus.Labels{
		"location": cw.Name,
		"lat":      strconv.FormatFloat(cw.Lat, 'f', 4, 64),
		"lon":      strconv.FormatFloat(cw.Lon, 'f', 4, 64),
		"country":  cw.Country,
	}).Set(1)
	return nil
}

// WeatherMetricsConverter — 11 dynamic gauges, all with "location" label
type WeatherMetricsConverter struct{}

func NewWeatherMetricsConverter() *WeatherMetricsConverter { return &WeatherMetricsConverter{} }

func (c *WeatherMetricsConverter) Match(name string) bool {
	return name == CurrentWeatherType
}

func (c *WeatherMetricsConverter) Convert(registry metric.Registry, data any) error {
	cw, ok := data.(CurrentWeather)
	if !ok {
		return fmt.Errorf("%w: %v", ErrInvalidDataType, reflect.TypeOf(data))
	}

	set := func(name, help string, value float64) {
		registry.GetOrCreateGaugeVec(name, help, []string{"location"}).
			WithLabelValues(cw.Name).Set(value)
	}

	set("temperature", "Current temperature in Celsius", cw.Temperature)
	set("feels_like", "Perceived temperature in Celsius", cw.FeelsLike)
	set("humidity", "Relative humidity in percent", cw.Humidity)
	set("pressure", "Atmospheric pressure in hPa", cw.Pressure)
	set("wind_speed", "Wind speed in m/s", cw.WindSpeed)
	set("wind_direction", "Wind direction in degrees", cw.WindDeg)
	set("cloud_coverage", "Cloudiness in percent", cw.Clouds)
	set("visibility", "Visibility in metres", cw.Visibility)
	set("uv_index", "UV index", cw.UVIndex)
	set("sunrise_timestamp", "Sunrise time as Unix timestamp", float64(cw.Sunrise))
	set("sunset_timestamp", "Sunset time as Unix timestamp", float64(cw.Sunset))
	return nil
}
