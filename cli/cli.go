package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/reeveci/reeve/buildinfo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getDefaultConfigFile() string {
	if configDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(configDir, "reeve", "reeve.toml")
	}

	return filepath.Join(".", ".reeve.toml")
}

var programName = os.Args[0]

var defaultConfigFile = getDefaultConfigFile()

const defaultAuthHeader = "Authorization"
const defaultAuthPrefix = "Bearer "

type Config struct {
	URL      string `mapstructure:"url"`
	Insecure bool   `mapstructure:"insecure"`
	Secret   string `mapstructure:"secret"`

	Auth struct {
		Header string `mapstructure:"header"`
		Prefix string `mapstructure:"prefix"`
	} `mapstructure:"auth"`
}

var configFile string
var config Config

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().StringVar(&configFile, "config", "", "Location of client config file (default \""+defaultConfigFile+"\")")

	rootCmd.Flags().String("url", "", "Reeve server URL")
	rootCmd.Flags().Bool("insecure", false, "Allow insecure TLS connections by skipping certificate verification")
	rootCmd.Flags().String("secret", "", "CLI secret")

	rootCmd.Flags().String("auth-header", defaultAuthHeader, "Authorization header")
	rootCmd.Flags().String("auth-prefix", defaultAuthPrefix, "Authorization prefix")

	viper.BindPFlag("auth.header", rootCmd.Flags().Lookup("auth-header"))
	viper.BindPFlag("auth.prefix", rootCmd.Flags().Lookup("auth-prefix"))
	viper.BindPFlags(rootCmd.Flags())
}

func initConfig() {
	viper.SetDefault("auth.header", defaultAuthHeader)
	viper.SetDefault("auth.prefix", defaultAuthPrefix)

	viper.SetEnvPrefix("REEVE_CLI")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if configFile == "" {
		configFile = os.Getenv("REEVE_CLI_CONFIG")
	}
	if configFile == "" {
		configFile = defaultConfigFile
	}
	if err := loadConfigFile(viper.GetViper()); err != nil {
		fmt.Fprintln(os.Stderr, "Cannot load config:", err)
		os.Exit(1)
	}

	if err := viper.Unmarshal(&config); err != nil {
		fmt.Fprintln(os.Stderr, "Cannot load config:", err)
		os.Exit(1)
	}
}

func Execute(buildInfo buildinfo.BuildInfo) {
	rootCmd.Version = buildInfo.Version

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
