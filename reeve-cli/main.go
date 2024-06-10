package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"

	"github.com/BurntSushi/toml"
	"golang.org/x/term"
)

var buildVersion = "development"

type Config struct {
	Secret string `toml:"secret"`
	URL    string `toml:"url"`
}

func main() {
	errLog := log.New(os.Stderr, "", 0)

	// Resolve default config file path
	var defaultConfigFile string
	if defaultConfigFile = os.Getenv("REEVECLI_CONFIG"); defaultConfigFile == "" {
		if configDir, err := os.UserConfigDir(); err == nil {
			defaultConfigFile = filepath.Join(configDir, "reeve")
		}
		if defaultConfigFile == "" {
			defaultConfigFile = "."
		}
		defaultConfigFile = filepath.Join(defaultConfigFile, ".reevecli")
	}

	// Parse CLI flags
	var version bool
	var configFile string
	var config Config
	var insecure bool
	var usage bool

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options...] <target> <method> [<arg> ...]\n\nConnection settings are stored in a config file, which is located in the users home directory by default and can be otherwise specified with either the REEVECLI_CONFIG environment variable or the --config command flag. If you do not want to store the configuration on disk, you can also use /dev/null for the config file.\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.BoolVar(&version, "version", false, "print build information and exit")
	flag.BoolVar(&version, "v", false, "print build information and exit (shorthand)")
	flag.StringVar(&configFile, "config", defaultConfigFile, "config file")
	flag.StringVar(&configFile, "c", defaultConfigFile, "config file (shorthand)")
	flag.BoolVar(&insecure, "insecure", false, "allow insecure TLS connections by skipping certificate verification")
	flag.BoolVar(&usage, "usage", false, "print available methods")

	flag.StringVar(&config.URL, "url", "", "reeve server URL")
	flag.StringVar(&config.URL, "u", "", "reeve server URL (shorthand)")
	flag.StringVar(&config.Secret, "secret", "", "secret (use - for stdin)")
	flag.StringVar(&config.Secret, "s", "", "secret (use - for stdin) (shorthand)")

	flag.Parse()

	if version {
		fmt.Printf("%s version %s\n", path.Base(os.Args[0]), buildVersion)
		return
	}

	// Read config file
	if config.URL == "" || config.Secret == "" {
		var parsedConfig Config
		_, err := toml.DecodeFile(configFile, &parsedConfig)
		if err == nil {
			if config.URL == "" {
				config.URL = parsedConfig.URL
			}
			if config.URL == parsedConfig.URL && config.Secret == "" {
				config.Secret = parsedConfig.Secret
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			errLog.Printf("error reading config file, skipping - %s\n", err)
		}
	}

	// Exit if URL is missing
	if config.URL == "" {
		errLog.Println("missing server URL")
		flag.Usage()
		os.Exit(1)
		return
	}

	// Input secret from STDIN
	if config.Secret == "" || config.Secret == "-" {
		read, err := readPassword(fmt.Sprintf("Please enter secret for %s: ", config.URL))
		if err != nil {
			errLog.Fatalf("error reading secret from stdin - %s\n", err)
			return
		}
		config.Secret = string(read)
	}

	// Write config file
	if err := os.MkdirAll(filepath.Dir(configFile), 0750); err != nil {
		errLog.Printf("error setting up config directory, skipping - %s\n", err)
	} else {
		f, err := os.Create(configFile)
		if err == nil {
			err = toml.NewEncoder(f).Encode(config)
		}
		if err != nil {
			errLog.Printf("error writing config file, skipping - %s\n", err)
		}
	}

	// Exit if secret is missing
	if config.Secret == "" {
		errLog.Fatalf("missing secret")
		return
	}

	// Create client
	auth := fmt.Sprintf("Bearer %s", config.Secret)
	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}}

	// Print usage
	if usage {
		flag.CommandLine.SetOutput(os.Stdout)
		flag.Usage()

		fmt.Print("\n\nfetching available methods... ")

		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/cli", config.URL), nil)
		if err != nil {
			errLog.Fatalf("creating HTTP request failed - %s\n", err)
			return
		}

		req.Header.Set("Authorization", auth)

		resp, err := client.Do(req)
		if err != nil {
			errLog.Fatalf("error - %s\n", err)
			return
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errorMessage, _ := io.ReadAll(resp.Body)
			errLog.Fatalf("error - status %v - %s\n", resp.StatusCode, string(errorMessage))
			return
		}

		if resp.Header.Get("Content-Type") != "application/json" {
			errLog.Fatalln("error - Content-Type header is not application/json")
			return
		}

		var serverUsage map[string]map[string]string
		err = json.NewDecoder(resp.Body).Decode(&serverUsage)
		if err != nil {
			errLog.Fatalf("error - received invalid usage - %s\n", err)
			return
		}

		if len(serverUsage) == 0 {
			errLog.Fatalln("the server does not provide any CLI methods")
			return
		}

		fmt.Printf("done\n\n")

		plugins := make([]string, 0, len(serverUsage))
		for plugin := range serverUsage {
			plugins = append(plugins, plugin)
		}
		sort.Strings(plugins)

		for _, plugin := range plugins {
			fmt.Printf("  %s\n", plugin)

			pluginMethods := serverUsage[plugin]
			methods := make([]string, 0, len(pluginMethods))
			for method := range pluginMethods {
				methods = append(methods, method)
			}
			sort.Strings(methods)

			for _, method := range methods {
				description := pluginMethods[method]
				fmt.Printf("        %s: %s\n", method, strings.ReplaceAll(description, "\n", "\n        "))
			}
		}

		return
	}

	// Execute method
	args := flag.Args()
	if len(args) < 2 {
		errLog.Println("missing arguments")
		flag.Usage()
		os.Exit(1)
		return
	}

	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(args[2:])
	if err != nil {
		errLog.Fatalf("encoding args failed - %s\n", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/cli?target=%s&method=%s", config.URL, args[0], args[1]), buffer)
	if err != nil {
		errLog.Fatalf("creating HTTP request failed - %s\n", err)
		return
	}

	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		errLog.Fatalf("executing method failed - %s\n", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorMessage, _ := io.ReadAll(resp.Body)
		errLog.Fatalf("executing method failed - status %v - %s\n", resp.StatusCode, string(errorMessage))
		return
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		errLog.Fatalf("reading response failed - %s", err)
		return
	}

	fmt.Println(string(result))
}

func readPassword(query string) ([]byte, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		fmt.Print(query)
		defer fmt.Println()
		return readPasswordTerm(fd)
	} else {
		return readPasswordDirect(os.Stdin)
	}
}

func readPasswordTerm(fd int) ([]byte, error) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	oldState, err := term.GetState(fd)
	if err != nil {
		return nil, err
	}
	defer term.Restore(fd, oldState)
	result := make(chan pw)
	go func() {
		p, err := term.ReadPassword(fd)
		result <- pw{text: p, err: err}
		close(result)
	}()
	select {
	case <-signals:
		return nil, fmt.Errorf("canceled by user")
	case password := <-result:
		return password.text, password.err
	}
}

type pw struct {
	text []byte
	err  error
}

func readPasswordDirect(reader io.Reader) ([]byte, error) {
	var buf [1]byte
	var ret []byte

	for {
		n, err := reader.Read(buf[:])
		if n > 0 {
			switch buf[0] {
			case '\b':
				if len(ret) > 0 {
					ret = ret[:len(ret)-1]
				}
			case '\n':
				if runtime.GOOS != "windows" {
					return ret, nil
				}
				// otherwise ignore \n
			case '\r':
				if runtime.GOOS == "windows" {
					return ret, nil
				}
				// otherwise ignore \r
			default:
				ret = append(ret, buf[0])
			}
			continue
		}
		if err != nil {
			if err == io.EOF && len(ret) > 0 {
				return ret, nil
			}
			return ret, err
		}
	}
}
