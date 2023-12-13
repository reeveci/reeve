package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/djherbis/stream"
	"github.com/reeveci/reeve-lib/schema"
	"github.com/reeveci/reeve-lib/streams"
	"github.com/reeveci/reeve/reeve-server/runtime"
)

func HandleWorkerLogs(runtime *runtime.Runtime) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			GetPosition(runtime, res, req)

		case http.MethodPost:
			WriteLogs(runtime, res, req)

		default:
			http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}
}

func GetPosition(runtime *runtime.Runtime, res http.ResponseWriter, req *http.Request) {
	if !checkWorkerToken(req, runtime.WorkerSecrets) {
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	q := req.URL.Query()

	workerGroup := q.Get("group")
	if workerGroup == "" {
		workerGroup = schema.DEFAULT_WORKER_GROUP
	}
	workerActivity, ok := runtime.Activity[workerGroup]
	if !ok {
		http.Error(res, fmt.Sprintf("invalid worker group %s", workerGroup), http.StatusBadRequest)
		return
	}

	activityID := q.Get("activity")
	if activityID == "" {
		http.Error(res, `missing required query parameter "activity"`, http.StatusBadRequest)
		return
	}

	status := workerActivity.Status(activityID)
	if status == nil {
		http.Error(res, fmt.Sprintf("invalid activity %s", activityID), http.StatusBadRequest)
		return
	}

	status.Lock()

	var response schema.WorkerLogsPositionResponse

	switch status.Status {
	case schema.STATUS_WAITING:
		response.Position = 0

	case schema.STATUS_RUNNING:
		reader, err := status.Logs.Reader()
		if err != nil {
			status.Unlock()
			http.Error(res, fmt.Sprintf("unable to get stream reader - %s", err), http.StatusInternalServerError)
			return
		}
		response.Position, _ = reader.Size()
		reader.Close()

	default:
		status.Unlock()
		http.Error(res, fmt.Sprintf("invalid activity %s", activityID), http.StatusBadRequest)
		return
	}

	status.ResetTimeout(workerActivity.Timeout)
	status.Unlock()

	res.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(res).Encode(response)
	if err != nil {
		http.Error(res, fmt.Sprintf("error encoding response - %s", err), http.StatusInternalServerError)
		return
	}
}

func WriteLogs(runtime *runtime.Runtime, res http.ResponseWriter, req *http.Request) {
	if !checkWorkerToken(req, runtime.WorkerSecrets) {
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	q := req.URL.Query()

	workerGroup := q.Get("group")
	if workerGroup == "" {
		workerGroup = schema.DEFAULT_WORKER_GROUP
	}
	workerActivity, ok := runtime.Activity[workerGroup]
	if !ok {
		http.Error(res, fmt.Sprintf("invalid worker group %s", workerGroup), http.StatusBadRequest)
		return
	}

	activityID := q.Get("activity")
	if activityID == "" {
		http.Error(res, `missing required query parameter "activity"`, http.StatusBadRequest)
		return
	}

	status := workerActivity.Status(activityID)
	if status == nil {
		http.Error(res, fmt.Sprintf("invalid activity %s", activityID), http.StatusBadRequest)
		return
	}

	status.Lock()
	var logs *streams.StreamProvider

	switch status.Status {
	case schema.STATUS_WAITING:
		status.ClearTimeout()
		logs = streams.NewStreamProvider(stream.NewMemStream())
		status.Logs = logs
		status.Status = schema.STATUS_RUNNING
		status.Unlock()

		workerActivity.NotifyUpdate(activityID)

	case schema.STATUS_RUNNING:
		status.ClearTimeout()
		logs = status.Logs.(*streams.StreamProvider)
		status.Unlock()

	default:
		status.Unlock()
		http.Error(res, fmt.Sprintf("invalid activity %s", activityID), http.StatusBadRequest)
		return
	}

	_, err := io.Copy(logs, req.Body)

	status.Lock()
	status.ResetTimeout(workerActivity.Timeout)
	status.Unlock()

	if err != nil {
		http.Error(res, fmt.Sprintf("error processing request body - %s", err), http.StatusInternalServerError)
		return
	}
}
