//go:build !windows

package server

const defaultConfigFile = "/etc/reeve/server.toml"
const defaultPluginDir = "/etc/reeve/plugins"
const defaultStateDir = "/var/lib/reeve/server"
const defaultUnixSocket = `/var/run/reeve.sock`
const defaultGroup = `reeve`

func initPlatform() {
	rootCmd.Flags().String("socket", defaultUnixSocket, "Location of the unix socket")
	rootCmd.Flags().String("group", defaultGroup, "Group for the unix socket")
}
