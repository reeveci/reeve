package server

import (
	"github.com/reeveci/reeve/buildinfo"
	"github.com/reeveci/reeve/server/legacy"
)

func Execute(buildInfo buildinfo.BuildInfo) {
	legacy.Execute(buildInfo)
}
