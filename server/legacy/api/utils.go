package api

import (
	"net/http"
	"strings"
)

const TOKEN_QUERY_PARAM = "token"

func checkMessageToken(req *http.Request, secrets map[string]bool) bool {
	token := req.Header.Get("Authorization")

	if token != "" {
		token = strings.TrimPrefix(token, "Bearer ")
		token = strings.TrimPrefix(token, "Basic ")
	} else {
		token = req.URL.Query().Get(TOKEN_QUERY_PARAM)
	}

	token = strings.TrimSpace(token)
	return len(token) > 0 && secrets[token]
}

func checkCLIToken(req *http.Request, secrets map[string]bool) bool {
	return checkBearerToken(req, secrets)
}

func checkWorkerToken(req *http.Request, secrets map[string]bool) bool {
	return checkBearerToken(req, secrets)
}

func checkBearerToken(req *http.Request, secrets map[string]bool) bool {
	token := req.Header.Get("Authorization")

	if !strings.HasPrefix(token, "Bearer ") {
		return false
	}

	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	return len(token) > 0 && secrets[token]
}
