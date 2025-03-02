package runner

import (
	"github.com/reeveci/reeve/buildinfo"
	"github.com/reeveci/reeve/runner/legacy"
)

func Execute(buildInfo buildinfo.BuildInfo) {
	legacy.Execute(buildInfo)
}
