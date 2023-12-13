package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/reeveci/reeve-lib/schema"
	"github.com/reeveci/reeve/reeve-server/runtime"
)

func HandleWorkerAck(runtime *runtime.Runtime) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if req.Header.Get("Content-type") != "application/json" {
			http.Error(res, "Content-Type header is not application/json", http.StatusUnsupportedMediaType)
			return
		}

		if !checkWorkerToken(req, runtime.WorkerSecrets) {
			http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		workerGroup := req.URL.Query().Get("group")
		if workerGroup == "" {
			workerGroup = schema.DEFAULT_WORKER_GROUP
		}
		queue, ok := runtime.WorkerQueues[workerGroup]
		if !ok {
			http.Error(res, fmt.Sprintf("invalid worker group %s", workerGroup), http.StatusBadRequest)
			return
		}

		queue.Contract.Consumer.Lock()

		var data schema.WorkerAckRequest
		decoder := json.NewDecoder(req.Body)
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&data)
		if err != nil {
			queue.Contract.Consumer.Unlock()
			http.Error(res, fmt.Sprintf("invalid request body - %s", err), http.StatusBadRequest)
			return
		}

		if data.Contract != queue.Contract.Contract || queue.Contract.IsCanceled() {
			queue.Contract.Consumer.Unlock()
			http.Error(res, fmt.Sprintf("invalid contract %s for worker group %s", data.Contract, workerGroup), http.StatusBadRequest)
			return
		}

		pipelineActivity := queue.Pop()
		queue.Contract.Finish()

		workerActivity := runtime.Activity[workerGroup]
		status := workerActivity.Status(pipelineActivity.ActivityID)
		if status == nil {
			http.Error(res, fmt.Sprintf("missing status for activity %s", pipelineActivity.ActivityID), http.StatusInternalServerError)
			return
		}

		status.Lock()
		status.Status = schema.STATUS_WAITING
		status.ResetTimeout(workerActivity.Timeout)
		status.Unlock()

		workerActivity.NotifyUpdate(pipelineActivity.ActivityID)
	}
}
