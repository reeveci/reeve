package server

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/reeveci/reeve/buildinfo"
	"github.com/reeveci/reeve/server/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var programName = os.Args[0]

const commonSettingEnvPrefix = "REEVE_COMMON_"

var pluginEnvNameRegex = regexp.MustCompile(`^REEVE_PLUGIN_(?P<Plugin>[a-zA-Z0-9]+)_(?P<Key>.+)$`)
var pluginEnvNameIPlugin = pluginEnvNameRegex.SubexpIndex("Plugin")
var pluginEnvNameIKey = pluginEnvNameRegex.SubexpIndex("Key")
var pluginCLINameRegex = regexp.MustCompile(`^(?P<Plugin>[a-zA-Z0-9]+)\.(?P<Key>.+)$`)
var pluginCLINameIPlugin = pluginCLINameRegex.SubexpIndex("Plugin")
var pluginCLINameIKey = pluginCLINameRegex.SubexpIndex("Key")

var configFile string
var cliCommonSettings map[string]string
var cliPluginSettings map[string]string

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().StringVar(&configFile, "config", "", "Location of the server config file (default \""+defaultConfigFile+"\")")

	rootCmd.Flags().String("plugin-dir", defaultPluginDir, "Location of the plugin directory")
	rootCmd.Flags().String("state-dir", defaultStateDir, "Location of the state directory")

	rootCmd.Flags().Int("http-port", 0, "API HTTP port")
	rootCmd.Flags().Int("https-port", 0, "API HTTPS port")

	rootCmd.Flags().String("tls-cert", "", "Location of the TLS certificate file for the HTTPS server")
	rootCmd.Flags().String("tls-key", "", "Location of the TLS key file for the HTTPS server")

	rootCmd.Flags().StringToStringVar(&cliCommonSettings, "common-setting", nil, "Configure a setting to be passed to all plugins (key=value)")
	rootCmd.Flags().StringToStringVar(&cliPluginSettings, "setting", nil, "Configure a setting to be passed to a specific plugin (plugin.key=value)")

	initPlatform()

	viper.BindPFlag("api.http-port", rootCmd.Flags().Lookup("http-port"))
	viper.BindPFlag("api.https-port", rootCmd.Flags().Lookup("https-port"))
	viper.BindPFlag("api.tls.cert-file", rootCmd.Flags().Lookup("tls-cert"))
	viper.BindPFlag("api.tls.key-file", rootCmd.Flags().Lookup("tls-key"))
	viper.BindPFlags(rootCmd.Flags())
}

func initConfig() {
	viper.SetDefault("plugin-dir", defaultPluginDir)
	viper.SetDefault("state-dir", defaultStateDir)

	viper.SetEnvPrefix("REEVE_SERVER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	if configFile == "" {
		configFile = os.Getenv("REEVE_SERVER_CONFIG")
	}
	if configFile == "" {
		configFile = defaultConfigFile
	}
	if err := loadConfigFile(viper.GetViper()); err != nil {
		fmt.Fprintln(os.Stderr, "Cannot load config:", err)
		os.Exit(1)
	}

	if err := viper.Unmarshal(&config.Config); err != nil {
		fmt.Fprintln(os.Stderr, "Cannot load config:", err)
		os.Exit(1)
	}

	loadPluginSettings()
}

func loadPluginSettings() {
	commonSettings := make(map[string]string, len(config.Config.Common))
	pluginSettings := make(map[string]map[string]string, len(config.Config.Plugins))

	// Config file
	for key, value := range config.Config.Common {
		commonSettings[strings.ToLower(key)] = value
	}
	for pluginName, pluginConfig := range config.Config.Plugins {
		settings := make(map[string]string, len(pluginConfig))
		pluginSettings[strings.ToLower(pluginName)] = settings
		for key, value := range pluginConfig {
			settings[strings.ToLower(key)] = value
		}
	}

	// Environment variables
	for _, env := range os.Environ() {
		envName := strings.Split(env, "=")[0]
		if strings.HasPrefix(envName, commonSettingEnvPrefix) && len(envName) > len(commonSettingEnvPrefix) {
			key := strings.TrimPrefix(envName, commonSettingEnvPrefix)
			commonSettings[strings.ToLower(key)] = os.Getenv(envName)
		}
		if matches := pluginEnvNameRegex.FindStringSubmatch(envName); len(matches) > pluginEnvNameIPlugin && len(matches) > pluginEnvNameIKey {
			pluginName := matches[pluginEnvNameIPlugin]
			key := matches[pluginEnvNameIKey]
			settings := pluginSettings[strings.ToLower(pluginName)]
			if settings == nil {
				settings = make(map[string]string)
				pluginSettings[strings.ToLower(pluginName)] = settings
			}
			settings[strings.ToLower(key)] = os.Getenv(envName)
		}
	}

	// CLI flags
	for key, value := range cliCommonSettings {
		commonSettings[key] = value
	}
	for name, value := range cliPluginSettings {
		if matches := pluginCLINameRegex.FindStringSubmatch(name); len(matches) > pluginCLINameIPlugin && len(matches) > pluginCLINameIKey {
			pluginName := matches[pluginCLINameIPlugin]
			key := matches[pluginCLINameIKey]
			settings := pluginSettings[strings.ToLower(pluginName)]
			if settings == nil {
				settings = make(map[string]string)
				pluginSettings[strings.ToLower(pluginName)] = settings
			}
			settings[strings.ToLower(key)] = value
		} else {
			fmt.Fprintf(os.Stderr, "Invalid setting \"%s\": Setting must be formatted as plugin.key=value\n", name)
			os.Exit(1)
		}
	}

	config.Config.Common = commonSettings
	config.Config.Plugins = pluginSettings
}

func Execute(buildInfo buildinfo.BuildInfo) {
	rootCmd.Version = buildInfo.Version

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
