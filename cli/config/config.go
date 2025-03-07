package config

type CLIConfig struct {
	URL      string `mapstructure:"url"`
	Insecure bool   `mapstructure:"insecure"`
	Secret   string `mapstructure:"secret"`

	Auth struct {
		Header string `mapstructure:"header"`
		Prefix string `mapstructure:"prefix"`
	} `mapstructure:"auth"`
}

var Config CLIConfig
