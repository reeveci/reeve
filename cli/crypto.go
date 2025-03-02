package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/reeveci/reeve-lib/crypto"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cryptoCmd)
}

var cryptoCmd = &cobra.Command{
	Use:                   "crypto",
	Short:                 "Cryptographic utility functions",
	DisableFlagsInUseLine: true,

	TraverseChildren: true,
}

var noTrim bool

func init() {
	cryptoCmd.AddCommand(hashCmd)

	hashCmd.Flags().BoolVarP(&noTrim, "no-trim", "T", false, "Do not remove surrounding whitespace from the input")
}

var hashCmd = &cobra.Command{
	Use:   "hash",
	Short: "Create a hash of the value provided by stdin",
	Long: `Create a hash of the value provided by stdin

Note that surrounding whitespace is removed from the input by default.
You can disable this behavior by specifying the -T switch.`,
	DisableFlagsInUseLine: true,

	Args: cobra.NoArgs,

	Run: func(cmd *cobra.Command, args []string) {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		raw := string(input)
		if !noTrim {
			raw = strings.TrimSpace(raw)
		}

		pw, err := crypto.Hash(raw)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println(pw)
	},
}
