package api

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/reeveci/reeve/server/log"
)

func handleSocketAPI(mux *http.ServeMux) {
	// mux.Handle...
}

func ServeSocketAPI(wg *sync.WaitGroup) {
	l, uri, err := getSocketListener()
	if err != nil {
		log.Error.Fatalf("error setting up socket API - %v\n", err)
		return
	}

	mux := http.NewServeMux()

	handleSocketAPI(mux)

	httpServer := &http.Server{
		ReadHeaderTimeout: 5 * time.Minute, // "G112: Potential Slowloris Attack (gosec)"; not a real concern for our use, so setting a long timeout.
	}
	httpServer.Handler = mux

	wg.Add(1)
	go func() {
		defer wg.Done()

		log.System.Printf("serving socket API at %s\n", uri)
		err = httpServer.Serve(l)
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error.Fatalf("socket server failed - %s\n", err)
		}
	}()
}
