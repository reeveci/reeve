package server

import (
	"github.com/reeveci/reeve/server/server"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   programName,
	Short: "Reeve CI / CD Server",
	Long: `Reeve CI / CD Server

Most options can also be specified using environment variables, which need to be prefixed with 'REEVE_SERVER_', e.g. 'REEVE_SERVER_CONFIG=/path/to/config/file'.
Common plugin settings can be specified with the prefix 'REEVE_COMMON_', specific plugin settings with 'REEVE_PLUGIN_<PLUGIN_NAME>_'.`,
	DisableFlagsInUseLine: true,

	Version: "dev",

	Args: cobra.NoArgs,

	Run: func(cmd *cobra.Command, args []string) {
		server.Execute()
	},
}
