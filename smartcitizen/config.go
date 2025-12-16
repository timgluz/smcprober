package smartcitizen

const (
	DefaultUsernameEnv = "SMARTCITIZEN_USERNAME"
	DefaultPasswordEnv = "SMARTCITIZEN_PASSWORD"
	DefaultTokenEnv    = "SMARTCITIZEN_TOKEN"

	DefaultEndpoint   = "https://api.smartcitizen.me"
	DefaultAPIVersion = "v0"
)

type Config struct {
	Endpoint   string `json:"endpoint"`
	APIVersion string `json:"api_version"`

	UsernameEnv string `json:"username_env"`
	PasswordEnv string `json:"password_env"`
	TokenEnv    string `json:"token_env"`
}

func (c *Config) ApplyDefaults() {
	if c.Endpoint == "" {
		c.Endpoint = DefaultEndpoint
	}

	if c.APIVersion == "" {
		c.APIVersion = DefaultAPIVersion
	}

	if c.UsernameEnv == "" {
		c.UsernameEnv = DefaultUsernameEnv
	}

	if c.PasswordEnv == "" {
		c.PasswordEnv = DefaultPasswordEnv
	}

	if c.TokenEnv == "" {
		c.TokenEnv = DefaultTokenEnv
	}
}
