package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/reeveci/reeve-lib/schema"
	"github.com/reeveci/reeve/reeve-server/runtime"
)

func HandleWorkerResult(runtime *runtime.Runtime) http.HandlerFunc {
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

		if !status.Running() {
			status.Unlock()
			http.Error(res, fmt.Sprintf("invalid activity %s", activityID), http.StatusBadRequest)
			return
		}

		decoder := json.NewDecoder(req.Body)
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&status.Result)
		if err != nil {
			status.Unlock()
			http.Error(res, fmt.Sprintf("invalid request body - %s", err), http.StatusBadRequest)
			return
		}

		status.ClearTimeout()

		if status.Result.Success {
			status.Status = schema.STATUS_SUCCESS
		} else {
			status.Status = schema.STATUS_FAILED
		}

		status.Unlock()

		workerActivity.NotifyUpdate(activityID)
	}
}
