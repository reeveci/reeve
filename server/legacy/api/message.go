package api

import (
	"fmt"
	"io"
	"net/http"

	"github.com/reeveci/reeve-lib/schema"
	"github.com/reeveci/reeve/server/legacy/runtime"
)

func HandleMessage(runtime *runtime.Runtime) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost && req.Method != http.MethodGet {
			http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if !checkMessageToken(req, runtime.MessageSecrets) {
			http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		var message schema.Message

		q := req.URL.Query()
		message.Target = q.Get("target")
		message.Options = make(map[string]string, len(q))
		for k, v := range q {
			switch k {
			case "target":
				message.Target = v[0]
			case TOKEN_QUERY_PARAM:
				// ignore
			default:
				message.Options[k] = v[0]
			}
		}

		switch message.Target {
		case "":
			http.Error(res, `missing required query parameter "target"`, http.StatusBadRequest)
			return

		case schema.BROADCAST_MESSAGE:

		default:
			if _, ok := runtime.PluginProvider.MessagePlugins[message.Target]; !ok {
				http.Error(res, "message target is not available", http.StatusUnprocessableEntity)
				return
			}
		}

		var err error
		message.Data, err = io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, fmt.Sprintf("error reading request body - %s", err), http.StatusBadRequest)
			return
		}

		runtime.MessageQueue.Push(schema.FullMessage{Message: message, Source: schema.MESSAGE_SOURCE_API})
	}
}
