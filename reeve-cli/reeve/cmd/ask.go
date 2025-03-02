package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(askCommand)

	askCommand.Flags().BoolP("list", "l", false, "List available commands")
}

var askCommand = &cobra.Command{
	Use:   "ask plugin command [args]...",
	Short: "Ask the server to do something",
	Long: `Ask the server to do something

Any Reeve plugin can register commands to be executed by the CLI.
Run '` + programName + ` ask -l' to get a list of available commands.`,
	DisableFlagsInUseLine: true,

	DisableFlagParsing: true,

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
			cmd.Help()
			return
		}

		if len(args) > 0 && (args[0] == "--list" || args[0] == "-l") {
			ListAvailableCommands()
			return
		}

		if len(args) == 1 || (len(args) > 1 && (args[1] == "--list" || args[1] == "-l")) {
			ListAvailablePluginCommands(args[0])
			return
		}

		if l := len(args); l < 2 {
			fmt.Fprintf(os.Stderr, "Error: requires at least 2 arg(s), only received %v\n%s\nrequires at least 2 arg(s), only received %v\n", l, cmd.UsageString(), l)
			os.Exit(1)
		}

		RunCommand(args[0], args[1], args[2:])
	},
}

func ListAvailableCommands() {
	usage := GetCLICommands()

	if len(usage) == 0 {
		fmt.Println("No commands available")
		return
	}

	plugins := make([]string, 0, len(usage))
	for plugin := range usage {
		plugins = append(plugins, plugin)
	}
	sort.Strings(plugins)

	for _, plugin := range plugins {
		ListPluginCommands(plugin, usage[plugin])
	}
}

func ListAvailablePluginCommands(plugin string) {
	if plugin == "" {
		fmt.Fprintln(os.Stderr, "Missing plugin")
		os.Exit(1)
	}

	usage := GetCLICommands()

	if len(usage[plugin]) == 0 {
		fmt.Println("No commands available")
		return
	}

	ListPluginCommands(plugin, usage[plugin])
}

func ListPluginCommands(plugin string, pluginCommands map[string]string) {
	fmt.Printf("  %s\n", plugin)

	commands := make([]string, 0, len(pluginCommands))
	n := 0
	for command := range pluginCommands {
		commands = append(commands, command)
		n = max(n, len(command))
	}
	sort.Strings(commands)

	for _, command := range commands {
		description := pluginCommands[command]
		fmt.Printf("        %-"+strconv.Itoa(n)+"s   %s\n", command, strings.ReplaceAll(description, "\n", "\n        "))
	}
}

func RunCommand(plugin, command string, args []string) {
	result := ExecuteCommand(plugin, command, args)

	fmt.Println(result)
}
