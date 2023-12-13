package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/reeveci/reeve-lib/schema"
	"github.com/reeveci/reeve/reeve-server/runtime"
)

func HandleWorkerQueue(runtime *runtime.Runtime) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
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

		queue.Contract.Lock()

		pipelineActivity := queue.Get()

		select {
		case <-req.Context().Done():
			queue.Contract.Unlock()
			http.Error(res, "connection error", http.StatusInternalServerError)
			return
		default:
		}

		contract := queue.Contract.Next(runtime.QueueTimeout, func(contract string) {
			runtime.Status <- []string{fmt.Sprintf("[%s|%s: %s] contract timed out", workerGroup, pipelineActivity.ActivityID, pipelineActivity.Name)}
		})

		res.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(res).Encode(schema.WorkerQueueResponse{
			Contract: contract,
			Activity: pipelineActivity.ActivityID,
			Pipeline: pipelineActivity.Pipeline,
		})
		if err != nil {
			queue.Contract.Cancel()
			http.Error(res, fmt.Sprintf("error encoding pipeline - %s", err), http.StatusInternalServerError)
			return
		}
	}
}
