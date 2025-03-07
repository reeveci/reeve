package api

import (
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/reeveci/reeve/server/config"
	"github.com/reeveci/reeve/server/log"
)

func handleRestAPI(mux *http.ServeMux) {
	// mux.Handle...
}

func ServeRestAPI(wg *sync.WaitGroup) {
	mux := http.NewServeMux()

	handleSocketAPI(mux)

	var n int

	if config.Config.Api.HttpPort > 0 {
		n += 1
		wg.Add(1)
		go func() {
			defer wg.Done()

			log.System.Printf("serving REST API at http://0.0.0.0:%v\n", config.Config.Api.HttpPort)
			err := http.ListenAndServe(fmt.Sprintf(":%v", config.Config.Api.HttpPort), mux)
			if !errors.Is(err, http.ErrServerClosed) {
				log.Error.Fatalf("HTTP server failed - %s\n", err)
			}
		}()
	}

	if config.Config.Api.HttpsPort > 0 {
		if config.Config.Api.Tls.CertFile == "" || config.Config.Api.Tls.KeyFile == "" {
			log.Error.Fatalf("could not start HTTPS server - missing TLS configuration\n")
			return
		}

		n += 1
		wg.Add(1)
		go func() {
			defer wg.Done()

			log.System.Printf("serving REST API at https://0.0.0.0:%v\n", config.Config.Api.HttpsPort)
			err := http.ListenAndServeTLS(fmt.Sprintf(":%v", config.Config.Api.HttpsPort), config.Config.Api.Tls.CertFile, config.Config.Api.Tls.KeyFile, mux)
			if !errors.Is(err, http.ErrServerClosed) {
				log.Error.Fatalf("HTTPS server failed - %s\n", err)
			}
		}()
	}

	if n == 0 {
		log.Error.Fatalf("could not start REST server - neither a HTTP port nor a HTTPS port is configured\n")
		return
	}
}
