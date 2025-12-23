package ntfy

const (
	DefaultNtfyEndpoint    = "https://ntfy.sh"
	DefaultNtfyTopic       = "your-ntfy-topic"
	DefaultNtfyTokenEnvVar = "NTFY_TOKEN" // #nosec G101 -- This is an environment variable name, not a credential
)

type Config struct {
	Endpoint string `json:"endpoint"`
	Topic    string `json:"topic"`
	TokenEnv string `json:"token_env"`
}

func DefaultNtfyConfig() Config {
	return Config{
		Endpoint: DefaultNtfyEndpoint,
		Topic:    DefaultNtfyTopic,
		TokenEnv: DefaultNtfyTokenEnvVar,
	}
}

func (c *Config) ApplyDefaults() {
	if c.Endpoint == "" {
		c.Endpoint = DefaultNtfyEndpoint
	}

	if c.Topic == "" {
		c.Topic = DefaultNtfyTopic
	}

	if c.TokenEnv == "" {
		c.TokenEnv = DefaultNtfyTokenEnvVar
	}
}
