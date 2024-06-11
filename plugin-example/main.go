package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/reeveci/reeve-lib/plugin"
	"github.com/reeveci/reeve-lib/schema"
	"gopkg.in/yaml.v3"
)

const name = "example"

type Impl struct {
	logger hclog.Logger

	flag                         string
	taskDomains                  map[string]string
	trustedDomains, trustedTasks []string

	api plugin.ReeveAPI
}

func (i *Impl) Name() (string, error) {
	return name, nil
}

func (i *Impl) Register(settings map[string]string, api plugin.ReeveAPI) (capabilities plugin.Capabilities, err error) {
	i.logger.Info("GREETINGS FROM EXAMPLE PLUGIN!")

	i.api = api

	if strings.ToLower(settings["ENABLED"]) != "true" {
		return
	}

	i.flag = settings["FLAG"]
	taskDomains := strings.Fields(settings["TASK_DOMAINS"])
	i.taskDomains = make(map[string]string, len(taskDomains))
	for _, domain := range taskDomains {
		parts := strings.SplitN(domain, ":", 2)
		if len(parts) == 1 {
			i.taskDomains[parts[0]] = ""
		} else {
			i.taskDomains[parts[0]] = parts[1]
		}
	}
	i.trustedDomains = strings.Fields(settings["TRUSTED_DOMAINS"])
	i.trustedTasks = strings.Fields(settings["TRUSTED_TASKS"])

	capabilities.Message = true
	capabilities.Discover = true
	capabilities.Resolve = true
	capabilities.Notify = true

	capabilities.CLIMethods = map[string]string{
		"test": "test something",
	}
	return
}

func (i *Impl) Unregister() error {
	i.api.Close()

	return nil
}

func (i *Impl) Message(source string, message schema.Message) error {
	switch source {
	case schema.MESSAGE_SOURCE_SERVER:

	default:
		i.api.NotifyTriggers([]schema.Trigger{{"message": fmt.Sprintf(`"hello from plugin %s"`, name)}})
	}

	return nil
}

func (i *Impl) Discover(trigger schema.Trigger) ([]schema.Pipeline, error) {
	var test schema.PipelineDefinition
	if err := yaml.Unmarshal([]byte(TEST), &test); err != nil {
		return nil, fmt.Errorf("error parsing pipeline test: %s", err)
	}

	var test2 schema.PipelineDefinition
	if err := yaml.Unmarshal([]byte(TEST2), &test2); err != nil {
		return nil, fmt.Errorf("error parsing pipeline test2: %s", err)
	}

	return []schema.Pipeline{
		{
			PipelineDefinition: test,

			Env: map[string]schema.Env{
				"TEST": {Value: i.flag},
			},

			Facts: map[string]schema.Fact{
				"branch": {"main"},
			},
			TaskDomains:    i.taskDomains,
			TrustedDomains: i.trustedDomains,
			TrustedTasks:   i.trustedTasks,

			Setup: schema.Setup{
				RunConfig: schema.RunConfig{
					Task:    "docker",
					Command: []string{"sh", "-c", "echo hello, $test"},

					Params: map[string]schema.RawParam{
						"test": schema.EnvParam{
							Env: "TEST2",
						},
					},
				},
			},
		},

		{
			PipelineDefinition: test2,

			Env: map[string]schema.Env{
				"COMMAND": {Value: "echo test2 step"},
			},

			TaskDomains:    i.taskDomains,
			TrustedDomains: i.trustedDomains,
			TrustedTasks:   i.trustedTasks,

			Setup: schema.Setup{
				RunConfig: schema.RunConfig{
					Task:    "@trust/docker",
					Command: "docker ps",
				},
			},
		},
	}, nil
}

func (i *Impl) Resolve(env []string) (map[string]schema.Env, error) {
	result := make(map[string]schema.Env, len(env))
	for _, key := range env {
		result[key] = schema.Env{Value: i.flag}
	}
	return result, nil
}

func (i *Impl) Notify(status schema.PipelineStatus) error {
	defer status.Logs.Close()

	i.logger.Info(fmt.Sprintf("NOTIFY %s %s", status.ActivityID, status.Status))

	return nil
}

func (i *Impl) CLIMethod(method string, args []string) (string, error) {
	return "done with: " + method + " (" + strings.Join(args, ", ") + ")", nil
}

func main() {
	l := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	plugin.Serve(&plugin.PluginConfig{
		Plugin: &Impl{logger: l},

		Logger: l,
	})
}
