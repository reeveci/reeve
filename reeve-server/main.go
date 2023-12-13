package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/reeveci/reeve-lib/schema"
	"github.com/reeveci/reeve/reeve-server/api"
	"github.com/reeveci/reeve/reeve-server/runtime"
)

var buildVersion = "development"

func main() {
	var version bool

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options...]\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.BoolVar(&version, "version", false, "print build information and exit")
	flag.BoolVar(&version, "v", false, "print build information and exit (shorthand)")

	flag.Parse()

	if version {
		fmt.Printf("%s version %s\n", path.Base(os.Args[0]), buildVersion)
		return
	}

	fmt.Printf("welcome to reeve server version %s\n", buildVersion)

	procErrLog := log.New(os.Stderr, "", 0)

	runtime := runtime.GetRuntime()

	runtime.Log = log.New(os.Stdout, "*** ", 0)
	runtime.ProcLog = log.New(os.Stdout, "=== ", 0)
	runtime.ErrorLog = log.New(os.Stderr, "!!! ", 0)

	go runtime.LogStatus()

	err := runtime.LoadPlugins()
	if err != nil {
		procErrLog.Fatalf("error loading plugins - %s", err)
		return
	}
	defer runtime.PluginProvider.Close()

	if len(runtime.PluginProvider.DiscoverPlugins) == 0 {
		procErrLog.Fatalf("no discover plugins loaded")
		return
	}

	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signals
		procErrLog.Printf("received %s signal\n", sig)
		runtime.PluginProvider.Close()
		os.Exit(1)
	}()

	go HandleMessageQueues(runtime)
	go HandleTriggerQueue(runtime)
	go HandleNotifyQueue(runtime)

	runtime.MessageQueue.Push(schema.FullMessage{
		Message: schema.BroadcastMessage(map[string]string{"event": schema.EVENT_STARTUP_COMPLETE}, nil),
		Source:  schema.MESSAGE_SOURCE_SERVER,
	})

	api.ServeAPI(runtime)
}
