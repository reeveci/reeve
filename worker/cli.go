package worker

import (
	"github.com/reeveci/reeve/buildinfo"
	"github.com/reeveci/reeve/worker/legacy"
)

func Execute(buildInfo buildinfo.BuildInfo) {
	legacy.Execute(buildInfo)
}
