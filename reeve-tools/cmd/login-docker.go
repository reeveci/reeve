package cmd

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(loginDockerCmd)
}

var loginDockerCmd = &cobra.Command{
	Use:                   "login-docker user:password@registry...",
	Short:                 "Login to docker registries",
	DisableFlagsInUseLine: true,

	Args: cobra.MinimumNArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		for _, registry := range args {
			u, err := url.Parse("//" + registry)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing Docker registry - %s\n", err)
				os.Exit(1)
			}

			host := u.Host
			if host == "" {
				fmt.Fprintln(os.Stderr, "Error parsing Docker registry - missing host")
				os.Exit(1)
			}

			if u.Scheme != "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
				fmt.Fprintln(os.Stderr, "Error parsing Docker registry - invalid registry identifier")
				os.Exit(1)
			}

			fmt.Printf("Logging into Docker registry %s\n", host)

			user := u.User.Username()
			password, _ := u.User.Password()
			if user == "" || password == "" {
				fmt.Fprintf(os.Stderr, "Error logging into Docker registry %s - missing credentials\n", host)
				os.Exit(1)
			}

			cmd := exec.Command("docker", "login", "-u", user, "--password-stdin", host)
			cmd.Stdin = strings.NewReader(password)
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error logging into Docker registry %s - %s\n", host, err)
				os.Exit(1)
			}
		}
	},
}
