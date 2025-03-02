package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   programName,
	Short: "Reeve CI / CD Command Line Tools",
	Long: `Reeve CI / CD Command Line Tools

Most options can also be specified using environment variables, which need to be prefixed with 'REEVE_CLI_', e.g. 'REEVE_CLI_CONFIG=/path/to/config/file'.`,
	DisableFlagsInUseLine: true,

	Version: "dev",

	TraverseChildren: true,
}
