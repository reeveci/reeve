package log

import (
	"log"
	"os"
)

var System = log.New(os.Stderr, "=== ", 0)
var Error = log.New(os.Stderr, "!!! ", 0)
var Main = log.New(os.Stderr, "*** ", 0)
