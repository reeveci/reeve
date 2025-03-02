package buildinfo

import (
	_ "embed"
)

//go:embed VERSION
var BuildVersion string
