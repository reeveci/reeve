package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/djherbis/stream"
	"github.com/reeveci/reeve-lib/exe"
	"github.com/reeveci/reeve-lib/schema"
)

var buildVersion = "development"

type workerQueueResponse struct {
	Contract string          `json:"contract"`
	Activity string          `json:"activity"`
	Pipeline json.RawMessage `json:"pipeline"`
}

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

	fmt.Printf("welcome to reeve worker version %s\n", buildVersion)

	procErrLog := log.New(os.Stderr, "", 0)
	procLog := log.New(os.Stdout, "*** ", 0)

	apiUrl := exe.GetEnvDef("REEVE_SERVER_API", "http://localhost:9080")
	apiUrl = strings.TrimSuffix(apiUrl, "/")

	workerSecret := exe.GetEnvDef("REEVE_WORKER_SECRET", "")
	if workerSecret == "" {
		procErrLog.Fatalln("missing REEVE_WORKER_SECRET environment variable")
		return
	}

	workerGroup := exe.GetEnvDef("REEVE_WORKER_GROUP", schema.DEFAULT_WORKER_GROUP)
	runnerCommand := exe.GetEnvDef("REEVE_RUNNER_COMMAND", "reeve-runner")

	authHeader := exe.GetEnvDef("REEVE_WORKER_AUTH_HEADER", "Authorization")
	authPrefix := exe.GetEnvDef("REEVE_WORKER_AUTH_PREFIX", "Bearer ")
	auth := strings.TrimSpace(authPrefix + workerSecret)
	client := &http.Client{}

	procLog.Printf("connecting to %s", apiUrl)

	for {
		// Get message from worker queue
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/worker/queue?group=%s", apiUrl, workerGroup), nil)
		if err != nil {
			procErrLog.Fatalf("creating HTTP request failed - %s\n", err)
			return
		}

		req.Header.Set(authHeader, auth)

		resp, err := client.Do(req)
		if err != nil {
			procErrLog.Fatalf("fetching message from queue failed - %s\n", err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			errorMessage, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			procErrLog.Fatalf("fetching message from queue failed - status %v - %s\n", resp.StatusCode, string(errorMessage))
			return
		}

		if resp.Header.Get("Content-Type") != "application/json" {
			resp.Body.Close()
			procErrLog.Fatalln("fetching message from queue failed - Content-Type header is not application/json")
			return
		}

		var message workerQueueResponse
		err = json.NewDecoder(resp.Body).Decode(&message)
		resp.Body.Close()
		if err != nil {
			procErrLog.Printf("received invalid message from queue - %s\n", err)
			continue
		}

		// Send acknowledgement
		buffer := new(bytes.Buffer)
		err = json.NewEncoder(buffer).Encode(schema.WorkerAckRequest{
			Contract: message.Contract,
		})
		if err != nil {
			procErrLog.Fatalf("encoding acknowledgement failed - %s\n", err)
			return
		}

		req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/worker/ack?group=%s", apiUrl, workerGroup), buffer)
		if err != nil {
			procErrLog.Fatalf("creating HTTP request failed - %s\n", err)
			return
		}

		req.Header.Set(authHeader, auth)
		req.Header.Set("Content-Type", "application/json")

		resp, err = client.Do(req)
		if err != nil {
			procErrLog.Printf("sending acknowledgement failed - %s\n", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			errorMessage, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			procErrLog.Printf("sending acknowledgement failed - status %v - %s\n", resp.StatusCode, string(errorMessage))
			continue
		}

		resp.Body.Close()

		procLog.Println("starting pipeline execution")

		// Create runner output stream
		stream := stream.NewMemStream()
		var wg sync.WaitGroup
		wg.Add(2)

		// Send logs to stdout
		go func() {
			defer wg.Done()

			reader, err := stream.NextReader()
			if err != nil {
				procErrLog.Printf("reading pipeline logs failed - %s\n", err)
				return
			}

			defer reader.Close()

			_, err = io.Copy(os.Stdout, reader)
			if err != nil {
				procErrLog.Printf("reading pipeline logs failed - %s\n", err)
			}
		}()

		// Send logs to server
		go func() {
			defer wg.Done()

			SendLogs(stream, client, authHeader, auth, apiUrl, workerGroup, message.Activity, procLog, procErrLog)
		}()

		// Execute pipeline runner
		cmd := exec.Command(runnerCommand)
		cmd.Stdin = bytes.NewReader(message.Pipeline)
		cmd.Stdout = stream
		cmd.Stderr = stream
		cmd.Env = os.Environ()
		err = cmd.Run()

		stream.Close()
		wg.Wait()

		// Send pipeline result
		var result schema.PipelineResult

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				result.ExitCode = exitError.ExitCode()
			} else {
				result.ExitCode = 99
			}

			procLog.Printf("pipeline execution failed - %s\n", err)
		} else {
			result.Success = true

			procLog.Println("pipeline execution finished")
		}

		buffer = new(bytes.Buffer)
		err = json.NewEncoder(buffer).Encode(result)
		if err != nil {
			procErrLog.Fatalf("encoding pipeline result failed - %s\n", err)
			return
		}

		req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/worker/result?group=%s&activity=%s", apiUrl, workerGroup, message.Activity), buffer)
		if err != nil {
			procErrLog.Fatalf("creating HTTP request failed - %s\n", err)
			return
		}

		req.Header.Set(authHeader, auth)
		req.Header.Set("Content-Type", "application/json")

		resp, err = client.Do(req)
		if err != nil {
			procErrLog.Printf("sending pipeline result failed - %s\n", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			errorMessage, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			procErrLog.Printf("sending pipeline result failed - status %v - %s\n", resp.StatusCode, string(errorMessage))
			continue
		}

		resp.Body.Close()
	}
}

func SendLogs(stream *stream.Stream, client *http.Client, authHeader, auth, apiUrl, workerGroup, activity string, procLog, errorLog *log.Logger) {
	firstTry := true

	for {
		reader, err := stream.NextReader()
		if err != nil {
			errorLog.Printf("loading stdout stream reader failed - %s\n", err)
			return
		}

		if firstTry {
			firstTry = false
		} else {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/worker/logs?group=%s&activity=%s", apiUrl, workerGroup, activity), nil)
			if err != nil {
				reader.Close()
				errorLog.Printf("creating HTTP request failed - %s\n", err)
				return
			}

			req.Header.Set(authHeader, auth)

			resp, err := client.Do(req)
			if err != nil {
				reader.Close()
				errorLog.Printf("fetching pipeline log position failed, retrying - %s\n", err)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				reader.Close()
				errorMessage, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				errorLog.Printf("fetching pipeline log position failed - status %v - %s\n", resp.StatusCode, string(errorMessage))
				return
			}

			if resp.Header.Get("Content-Type") != "application/json" {
				reader.Close()
				resp.Body.Close()
				errorLog.Println("fetching pipeline log position failed - Content-Type header is not application/json")
				return
			}

			var message schema.WorkerLogsPositionResponse
			err = json.NewDecoder(resp.Body).Decode(&message)
			resp.Body.Close()
			if err != nil {
				reader.Close()
				errorLog.Printf("received invalid pipeline log position response - %s\n", err)
				return
			}

			reader.Seek(message.Position, io.SeekStart)
		}

		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/worker/logs?group=%s&activity=%s", apiUrl, workerGroup, activity), reader)
		if err != nil {
			reader.Close()
			errorLog.Printf("creating HTTP request failed - %s\n", err)
			return
		}

		req.Header.Set(authHeader, auth)

		resp, err := client.Do(req)
		if err != nil {
			errorLog.Printf("sending pipeline logs failed, retrying - %s\n", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			errorMessage, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode >= 500 {
				errorLog.Printf("sending pipeline logs failed, retrying - status %v - %s\n", resp.StatusCode, string(errorMessage))
				continue
			}

			errorLog.Printf("sending pipeline logs failed - status %v - %s\n", resp.StatusCode, string(errorMessage))
			return
		}

		resp.Body.Close()
		procLog.Println("done sending pipeline logs")
		return
	}
}
