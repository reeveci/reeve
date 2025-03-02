package cmd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	gourl "net/url"
	"os"
	"strings"
)

func PrepareRequest() (url string, authHeader, auth string, client *http.Client) {
	if config.URL == "" {
		fmt.Fprintln(os.Stderr, "Missing server URL")
		os.Exit(1)
	}
	url = config.URL

	authHeader = config.Auth.Header

	if config.Secret == "" {
		fmt.Fprintln(os.Stderr, "Missing secret")
		os.Exit(1)
	}
	auth = strings.TrimSpace(config.Auth.Prefix + config.Secret)

	client = &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.Insecure},
	}}
	return
}

func GetCLICommands() map[string]map[string]string {
	url, authHeader, auth, client := PrepareRequest()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/cli", url), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	req.Header.Set(authHeader, auth)

	res, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		errorMessage, _ := io.ReadAll(res.Body)
		fmt.Fprintf(os.Stderr, "Error: status %v - %s\n", res.StatusCode, string(errorMessage))
		os.Exit(1)
	}

	if contentType := res.Header.Get("Content-Type"); contentType != "application/json" {
		fmt.Fprintf(os.Stderr, "Error: Content-Type is not application/json, but %s\n", contentType)
		os.Exit(1)
	}

	var result map[string]map[string]string
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return result
}

func ExecuteCommand(plugin, command string, args []string) string {
	if plugin == "" {
		fmt.Fprintln(os.Stderr, "Missing plugin")
		os.Exit(1)
	}

	if command == "" {
		fmt.Fprintln(os.Stderr, "Missing command")
		os.Exit(1)
	}

	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	url, authHeader, auth, client := PrepareRequest()

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/cli?target=%s&method=%s", url, gourl.QueryEscape(plugin), gourl.QueryEscape(command)), buffer)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	req.Header.Set(authHeader, auth)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		errorMessage, _ := io.ReadAll(res.Body)
		fmt.Fprintf(os.Stderr, "Error: status %v - %s\n", res.StatusCode, string(errorMessage))
		os.Exit(1)
	}

	result, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return string(result)
}
