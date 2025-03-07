package config

type ServerConfig struct {
	PluginDir string `mapstructure:"plugin-dir"`
	StateDir  string `mapstructure:"state-dir"`
	Socket    string `mapstructure:"socket"`
	Pipe      string `mapstructure:"pipe"`
	Group     string `mapstructure:"group"`

	Api struct {
		HttpPort  int `mapstructure:"http-port"`
		HttpsPort int `mapstructure:"https-port"`

		Tls struct {
			CertFile string `mapstructure:"cert-file"`
			KeyFile  string `mapstructure:"key-file"`
		} `mapstructure:"tls"`
	} `mapstructure:"api"`

	Common  map[string]string            `mapstructure:"common"`
	Plugins map[string]map[string]string `mapstructure:"plugins"`
}

var Config ServerConfig
