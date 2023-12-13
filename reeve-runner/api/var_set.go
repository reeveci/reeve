package api

import (
	"net/http"

	"github.com/reeveci/reeve-lib/schema"
	"github.com/reeveci/reeve/reeve-runner/runtime"
)

func HandleVarSet(runtime *runtime.Runtime) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			q := req.URL.Query()
			key := q.Get("key")
			if key == "" {
				http.Error(res, `missing required query parameter "key"`, http.StatusBadRequest)
				return
			}

			v := q.Get("value")

			runtime.VarLock.Lock()
			runtime.Vars[key] = schema.Var(v)
			runtime.VarLock.Unlock()

		default:
			http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}
