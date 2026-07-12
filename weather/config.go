package weather

const (
	DefaultEndpoint       = "https://api.openweathermap.org"
	DefaultAPIPath        = "data/4.0"
	DefaultTokenEnv       = "OPENWEATHERMAP_TOKEN"
	DefaultNamespace      = "weather"
	DefaultScrapeInterval = 1800

	DefaultClimateEndpoint           = "https://archive-api.open-meteo.com"
	DefaultClimateAPIPath            = "v1/archive"
	DefaultClimateNormScrapeInterval = 86400 // norms barely change day to day — refetch once daily
	DefaultClimateNormalYears        = 30    // WMO-standard climate normal period length
	DefaultClimateNormalWindowDays   = 3     // +/- days around the target day-of-year to widen the sample
)

type Location struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

type Config struct {
	Endpoint  string     `json:"endpoint"`
	APIPath   string     `json:"api_path"`   // path segment, e.g. "data/4.0" — not a semver
	TokenEnv  string     `json:"token_env"`
	Locations []Location `json:"locations"`

	ClimateEndpoint         string `json:"climate_endpoint"`
	ClimateAPIPath          string `json:"climate_api_path"`
	ClimateNormalYears      int    `json:"climate_normal_years"`
	ClimateNormalWindowDays int    `json:"climate_normal_window_days"`
}

func (c *Config) ApplyDefaults() {
	// Fill zero-value string fields from Default* consts
	if c.Endpoint == "" {
		c.Endpoint = DefaultEndpoint
	}
	if c.APIPath == "" {
		c.APIPath = DefaultAPIPath
	}
	if c.TokenEnv == "" {
		c.TokenEnv = DefaultTokenEnv
	}
	if c.ClimateEndpoint == "" {
		c.ClimateEndpoint = DefaultClimateEndpoint
	}
	if c.ClimateAPIPath == "" {
		c.ClimateAPIPath = DefaultClimateAPIPath
	}
	if c.ClimateNormalYears <= 0 {
		c.ClimateNormalYears = DefaultClimateNormalYears
	}
	if c.ClimateNormalWindowDays <= 0 {
		c.ClimateNormalWindowDays = DefaultClimateNormalWindowDays
	}
}
