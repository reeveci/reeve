package server

import (
	"encoding/json"
	"fmt"
	"os"

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
		fmt.Println("moin")

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(config); err != nil {
			fmt.Fprintln(os.Stderr, "Cannot encode config:", err)
			os.Exit(1)
		}
	},
}
