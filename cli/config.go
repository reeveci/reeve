package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var fileConfig map[string]any

func init() {
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage client configuration files",
	Long: `Manage client configuration files

Note that configuration can be overridden using environment variables or flags. This utility only manages the configuration file.`,
	DisableFlagsInUseLine: true,

	TraverseChildren: true,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		parser := viper.New()

		if err := loadConfigFile(parser); err != nil {
			fmt.Fprintln(os.Stderr, "Cannot load config:", err)
			os.Exit(1)
		}

		fileConfig = parser.AllSettings()
	},
}

func init() {
	configCmd.AddCommand(configListCmd)
}

var configListCmd = &cobra.Command{
	Use:                   "list",
	Short:                 "List configured configuration options",
	DisableFlagsInUseLine: true,

	Args: cobra.NoArgs,

	Run: func(cmd *cobra.Command, args []string) {
		if err := toml.NewEncoder(os.Stdout).Encode(fileConfig); err != nil {
			fmt.Fprintln(os.Stderr, "Cannot encode config:", err)
			os.Exit(1)
		}
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
}

var configGetCmd = &cobra.Command{
	Use:                   "get key",
	Short:                 "Get a configuration option",
	DisableFlagsInUseLine: true,

	Args: cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if name == "" {
			fmt.Fprintln(os.Stderr, "Missing key")
			os.Exit(1)
		}

		parts := strings.Split(name, ".")
		section := fileConfig
		for ; len(parts) > 1; parts = parts[1:] {
			sub := section[parts[0]]
			section, _ = sub.(map[string]any)
		}

		switch value := section[parts[0]].(type) {
		case bool, float64, int64, string:
			fmt.Println(value)

		case nil:
			fmt.Fprintf(os.Stderr, "The option '%s' is not set\n", name)
			os.Exit(1)

		default:
			os.Exit(1)
		}
	},
}

var forceString bool
var forceBoolean bool
var forceNumber bool

func init() {
	configCmd.AddCommand(configSetCmd)

	configSetCmd.Flags().BoolVar(&forceString, "string", false, "Force the value to be a string")
	configSetCmd.Flags().BoolVar(&forceBoolean, "boolean", false, "Force the value to be a boolean")
	configSetCmd.Flags().BoolVar(&forceNumber, "number", false, "Force the value to be a number")
}

var configSetCmd = &cobra.Command{
	Use:   "set key value",
	Short: "Set a configuration option",
	Long: `Set a configuration option

The key is a dot separated path, e.g. 'auth.header'.
By default, the value is parsed as a boolean if it is exactly 'true' or 'false', a number if it can be parsed as such, or otherwise as a string.`,
	DisableFlagsInUseLine: true,

	Args: cobra.ExactArgs(2),

	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if name == "" {
			fmt.Fprintln(os.Stderr, "Missing key")
			os.Exit(1)
		}

		dt := "auto"
		if forceString {
			if dt != "auto" {
				fmt.Fprintln(os.Stderr, "Only one type flag is allowed")
			}
			dt = "string"
		}
		if forceBoolean {
			if dt != "auto" {
				fmt.Fprintln(os.Stderr, "Only one type flag is allowed")
			}
			dt = "boolean"
		}
		if forceNumber {
			if dt != "auto" {
				fmt.Fprintln(os.Stderr, "Only one type flag is allowed")
			}
			dt = "number"
		}

		rawValue := args[1]
		var value any
		switch dt {
		case "string":
			value = rawValue

		case "boolean":
			var err error
			value, err = strconv.ParseBool(rawValue)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

		case "number":
			var err error
			value, err = strconv.ParseInt(rawValue, 10, 64)
			if err != nil {
				value, err = strconv.ParseFloat(rawValue, 64)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

		default:
			if rawValue == "true" || rawValue == "false" {
				value = rawValue == "true"
			} else if v, err := strconv.ParseInt(rawValue, 10, 64); err == nil {
				value = v
			} else if v, err := strconv.ParseFloat(rawValue, 64); err == nil {
				value = v
			} else {
				value = rawValue
			}
		}

		parts := strings.Split(name, ".")
		if fileConfig == nil {
			fileConfig = make(map[string]any, 1)
		}
		section := fileConfig
		path := make([]string, 0, len(parts))
		for ; len(parts) > 1; parts = parts[1:] {
			path = append(path, parts[0])
			sub, ok := section[parts[0]].(map[string]any)
			if !ok {
				if section[parts[0]] != nil {
					fmt.Fprintf(os.Stderr, "'%s' is not a section, you need to unset it before setting '%s'\n", strings.Join(path, "."), name)
					os.Exit(1)
				}
				sub = make(map[string]any, 1)
				section[parts[0]] = sub
			}
			section = sub
		}

		section[parts[0]] = value

		WriteConfig()
	},
}

func init() {
	configCmd.AddCommand(configUnsetCmd)
}

var configUnsetCmd = &cobra.Command{
	Use:                   "unset key",
	Short:                 "Unset a configuration option",
	DisableFlagsInUseLine: true,

	Args: cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if name == "" {
			fmt.Fprintln(os.Stderr, "Missing key")
			os.Exit(1)
		}

		parts := strings.Split(name, ".")
		section := fileConfig
		for ; len(parts) > 1; parts = parts[1:] {
			var ok bool
			section, ok = section[parts[0]].(map[string]any)
			if !ok {
				fmt.Fprintf(os.Stderr, "The option '%s' is not set\n", name)
				os.Exit(1)
			}
		}

		if _, ok := section[parts[0]]; !ok {
			fmt.Fprintf(os.Stderr, "The option '%s' is not set\n", name)
			os.Exit(1)
		}
		delete(section, parts[0])

		WriteConfig()
	},
}

func WriteConfig() {
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		fmt.Fprintln(os.Stderr, "Cannot write config file:", err)
		os.Exit(1)
	}

	encoder := viper.New()
	if err := encoder.MergeConfigMap(fileConfig); err != nil {
		fmt.Fprintln(os.Stderr, "Cannot write config file:", err)
		os.Exit(1)
	}

	setupConfigFile(encoder)

	if err := encoder.WriteConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "Cannot write config file:", err)
		os.Exit(1)
	}
}
