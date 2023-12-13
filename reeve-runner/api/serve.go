package api

import (
	"fmt"
	"net/http"

	"github.com/reeveci/reeve/reeve-runner/runtime"
)

func ServeAPI(runtime *runtime.Runtime) {
	// start http service for message queue and worker queue
	// check messageplugins before sending message into queue

	http.HandleFunc("/api/v1/var", HandleVar(runtime))
	http.HandleFunc("/api/v1/var/set", HandleVarSet(runtime))

	ServeHTTP(runtime)
}

func ServeHTTP(runtime *runtime.Runtime) {
	runtime.Log.Subsystem("api").Printf("serving runner API at %s\n", runtime.ApiUrl)
	err := http.ListenAndServe(fmt.Sprintf(":%s", runtime.APIPort), nil)
	runtime.ErrorLog.Subsystem("api").Printf("HTTP server exited - %s\n", err)
}
