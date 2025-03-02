package api

import (
	"fmt"
	"net/http"

	"github.com/reeveci/reeve/server/legacy/runtime"
)

func ServeAPI(runtime *runtime.Runtime) {
	// start http service for message queue and worker queue
	// check messageplugins before sending message into queue

	// Message API
	http.HandleFunc(runtime.PathPrefix+"/message", HandleMessage(runtime))

	// CLI API
	http.HandleFunc(runtime.PathPrefix+"/cli", HandleCLI(runtime))

	// Worker API
	http.HandleFunc(runtime.PathPrefix+"/worker/queue", HandleWorkerQueue(runtime))
	http.HandleFunc(runtime.PathPrefix+"/worker/ack", HandleWorkerAck(runtime))
	http.HandleFunc(runtime.PathPrefix+"/worker/logs", HandleWorkerLogs(runtime))
	http.HandleFunc(runtime.PathPrefix+"/worker/result", HandleWorkerResult(runtime))

	hasHTTP := runtime.HTTPPort != ""
	hasHTTPS := runtime.HTTPSPort != "" && runtime.TLSCert != "" && runtime.TLSKey != ""

	if !hasHTTP {
		ServeHTTPS(runtime)
		return
	}

	if hasHTTPS {
		go ServeHTTPS(runtime)
	}
	ServeHTTP(runtime)
}

func ServeHTTPS(runtime *runtime.Runtime) {
	runtime.ProcLog.Printf("serving API at https://localhost:%s\n", runtime.HTTPSPort)
	err := http.ListenAndServeTLS(fmt.Sprintf(":%s", runtime.HTTPSPort), runtime.TLSCert, runtime.TLSKey, nil)
	runtime.ProcLog.Printf("HTTPS server exited - %s\n", err)
}

func ServeHTTP(runtime *runtime.Runtime) {
	runtime.ProcLog.Printf("serving API at http://localhost:%s\n", runtime.HTTPPort)
	err := http.ListenAndServe(fmt.Sprintf(":%s", runtime.HTTPPort), nil)
	runtime.ProcLog.Printf("HTTP server exited - %s\n", err)
}
