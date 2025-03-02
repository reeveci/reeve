package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	buildinfo "github.com/reeveci/reeve-cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getDefaultConfigDir() string {
	if configDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(configDir, "reeve")
	}

	return "."
}

var programName = os.Args[0]

const configFile = ".reevecli"

var defaultConfigDir = getDefaultConfigDir()

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

var configDir string
var config Config

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().StringVar(&configDir, "config", "", "Location of client config files (default \""+defaultConfigDir+"\")")

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

	if configDir == "" {
		configDir = os.Getenv("REEVE_CLI_CONFIG")
	}
	if configDir == "" {
		configDir = getDefaultConfigDir()
	}
	viper.AddConfigPath(configDir)
	viper.SetConfigName(configFile)
	viper.SetConfigType("toml")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "Cannot load config:", err)
			os.Exit(1)
		}
	}

	if err := viper.Unmarshal(&config); err != nil {
		fmt.Fprintln(os.Stderr, "Cannot load config:", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   programName,
	Short: "Reeve CI / CD - Command Line Tools",
	Long: `Reeve CI / CD - Command Line Tools

Most options can also be specified using environment variables, which need to be prefixed with 'REEVE_CLI_', e.g. 'REEVE_CLI_CONFIG=/path/to/config'.`,
	DisableFlagsInUseLine: true,

	Version: buildinfo.BuildVersion,

	TraverseChildren: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
