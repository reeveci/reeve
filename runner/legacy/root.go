package legacy

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/reeveci/reeve-lib/logs"
	"github.com/reeveci/reeve/buildinfo"
	"github.com/reeveci/reeve/runner/legacy/api"
	"github.com/reeveci/reeve/runner/legacy/runtime"
)

func Execute(buildInfo buildinfo.BuildInfo) {
	var version bool

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options...]\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.BoolVar(&version, "version", false, "print build information and exit")
	flag.BoolVar(&version, "v", false, "print build information and exit (shorthand)")

	flag.Parse()

	if version {
		fmt.Printf("%s version %s\n", path.Base(os.Args[0]), buildInfo.Version)
		return
	}

	log := logs.NewDecorated(os.Stdout, "", logs.NewDefaultDecorator("", "%-18s >"))
	errorLog := logs.NewDecorated(os.Stderr, "", logs.NewDefaultDecorator(":!", "%-18s >"))

	runtime, err := runtime.GetRuntime()
	if err != nil {
		errorLog.Subsystem("init").Println(err)
		os.Exit(1)
		return
	}

	runtime.Log = log
	runtime.ErrorLog = errorLog

	// Parse pipeline
	decoder := json.NewDecoder(os.Stdin)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&runtime.Pipeline)
	if err != nil {
		errorLog.Subsystem("init").Printf("error parsing pipeline from stdin - %s\n", err)
		os.Exit(1)
		return
	}

	// Check for runtime signals
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signals
		errorLog.Subsystem("signal").Printf("received %s signal\n", sig)
		runtime.Cancel()
	}()

	// Serve API
	go api.ServeAPI(runtime)

	// Run pipeline
	success := runtime.Run()

	if !success {
		os.Exit(2)
	}
}
