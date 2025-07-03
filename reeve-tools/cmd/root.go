package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var programName = os.Args[0]

var rootCmd = &cobra.Command{
	Use:                   programName,
	Short:                 "Reeve CI / CD - Step Tools",
	DisableFlagsInUseLine: true,

	TraverseChildren: true,
}

func Execute(buildVersion string) {
	rootCmd.Version = buildVersion

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
