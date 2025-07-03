package main

import "github.com/reeveci/reeve/reeve-tools/cmd"

var buildVersion = "development"

func main() {
	cmd.Execute(buildVersion)
}
