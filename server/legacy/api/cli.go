package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/reeveci/reeve/server/legacy/runtime"
)

func HandleCLI(runtime *runtime.Runtime) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			GetCLIUsage(runtime, res, req)

		case http.MethodPost:
			SendCLIMethod(runtime, res, req)

		default:
			http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}
}

func GetCLIUsage(runtime *runtime.Runtime, res http.ResponseWriter, req *http.Request) {
	if !checkCLIToken(req, runtime.CLISecrets) {
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	result := make(map[string]map[string]string, len(runtime.PluginProvider.CLIPlugins))

	for name, plugin := range runtime.PluginProvider.CLIPlugins {
		result[name] = plugin.CLIMethods
	}

	res.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(res).Encode(result)
	if err != nil {
		http.Error(res, fmt.Sprintf("error encoding CLI usage - %s", err), http.StatusInternalServerError)
		return
	}
}

func SendCLIMethod(runtime *runtime.Runtime, res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-type") != "application/json" {
		http.Error(res, "Content-Type header is not application/json", http.StatusUnsupportedMediaType)
		return
	}

	if !checkCLIToken(req, runtime.CLISecrets) {
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	q := req.URL.Query()

	target := q.Get("target")
	if target == "" {
		http.Error(res, `missing required query parameter "target"`, http.StatusBadRequest)
		return
	}
	plugin, ok := runtime.PluginProvider.CLIPlugins[target]
	if !ok {
		http.Error(res, "CLI target is not available", http.StatusUnprocessableEntity)
		return
	}

	method := q.Get("method")
	if method == "" {
		http.Error(res, `missing required query parameter "method"`, http.StatusBadRequest)
		return
	}
	if _, ok = plugin.CLIMethods[method]; !ok {
		http.Error(res, fmt.Sprintf("unavailable CLI method %s for target %s", method, target), http.StatusBadRequest)
		return
	}

	var args []string
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&args)
	if err != nil {
		http.Error(res, fmt.Sprintf("invalid request body - %s", err), http.StatusBadRequest)
		return
	}

	result, err := plugin.CLIMethod(method, args)
	if err != nil {
		http.Error(res, fmt.Sprintf("executing CLI method %s for target %s failed - %s\n", method, target, err), http.StatusInternalServerError)
		return
	}

	res.Write([]byte(result))
}
