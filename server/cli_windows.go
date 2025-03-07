//go:build windows

package server

const defaultConfigFile = `C:\ProgramData\reeve\config\server.toml`
const defaultPluginDir = `C:\ProgramData\reeve\config\plugins`
const defaultStateDir = `C:\ProgramData\reeve\server`
const defaultNamedPipe = `//./pipe/reeve_server`
const defaultGroup = ""

func initPlatform() {
	rootCmd.Flags().String("pipe", defaultNamedPipe, "Location of the named pipe")
	rootCmd.Flags().String("group", defaultGroup, "Users or groups that can access the named pipe")
}
