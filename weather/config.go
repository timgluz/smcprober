package weather

const (
	DefaultEndpoint       = "https://api.openweathermap.org"
	DefaultAPIPath        = "data/4.0"
	DefaultTokenEnv       = "OPENWEATHERMAP_TOKEN"
	DefaultNamespace      = "weather"
	DefaultScrapeInterval = 1800
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
}
