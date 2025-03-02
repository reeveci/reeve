package main

import (
	"github.com/reeveci/reeve/buildinfo"
	"github.com/reeveci/reeve/server"

	_ "embed"
)

//go:embed VERSION
var BuildVersion string

func main() {
	buildInfo := buildinfo.BuildInfo{
		Version: BuildVersion,
	}

	server.Execute(buildInfo)
}
