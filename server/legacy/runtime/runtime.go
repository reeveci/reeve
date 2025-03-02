package runtime

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/reeveci/reeve-lib/exe"
	"github.com/reeveci/reeve-lib/filter"
	"github.com/reeveci/reeve-lib/queue"
	"github.com/reeveci/reeve-lib/schema"
	"github.com/reeveci/reeve/server/legacy/activity"
)

const TIMEOUT_QUEUE = 1 * time.Minute
const TIMEOUT_ACTIVITY = 2 * time.Minute

type ContractQueue[T any] struct {
	queue.Queue[T]
	Contract Contract
}

type Runtime struct {
	PluginDirectory     string
	HTTPPort, HTTPSPort string
	PathPrefix          string
	TLSCert, TLSKey     string

	Log, ProcLog, ErrorLog *log.Logger

	MessageSecrets map[string]bool
	CLISecrets     map[string]bool
	WorkerSecrets  map[string]bool
	WorkerGroups   map[string]bool

	MessageQueue queue.Queue[schema.FullMessage]
	StatusQueue  queue.Queue[schema.PipelineStatus]
	TriggerQueue queue.Queue[schema.Trigger]
	NotifyQueue  queue.Queue[schema.PipelineStatus]

	MessageQueues map[string]queue.Queue[schema.FullMessage]
	WorkerQueues  map[string]*ContractQueue[activity.PipelineActivity]
	Activity      map[string]*activity.RuntimeActivity

	QueueTimeout time.Duration

	PluginProvider PluginProvider

	Status chan []string
}

func GetRuntime() *Runtime {
	runtime := Runtime{
		PluginDirectory: exe.GetEnvDef("REEVE_PLUGIN_DIRECTORY", "./plugins"),
		HTTPPort:        exe.GetEnvDef("REEVE_HTTP_PORT", ""),
		HTTPSPort:       exe.GetEnvDef("REEVE_HTTPS_PORT", ""),
		PathPrefix:      "/api/v1",
		TLSCert:         exe.GetEnvDef("REEVE_TLS_CERT_FILE", ""),
		TLSKey:          exe.GetEnvDef("REEVE_TLS_KEY_FILE", ""),

		MessageSecrets: exe.GetEnvFieldMap("REEVE_MESSAGE_SECRETS", ""),
		CLISecrets:     exe.GetEnvFieldMap("REEVE_CLI_SECRETS", ""),
		WorkerSecrets:  exe.GetEnvFieldMap("REEVE_WORKER_SECRETS", ""),
		WorkerGroups:   exe.GetEnvFieldMap("REEVE_WORKER_GROUPS", ""),

		MessageQueue: queue.Blocked(queue.NewQueue[schema.FullMessage]()),
		TriggerQueue: queue.Blocked(queue.NewQueue[schema.Trigger]()),
		NotifyQueue:  queue.Blocked(queue.NewQueue[schema.PipelineStatus]()),

		QueueTimeout: TIMEOUT_QUEUE,

		Status: make(chan []string, 20),
	}

	if runtime.HTTPPort == "" && (runtime.HTTPSPort == "" || runtime.TLSCert == "" || runtime.TLSKey == "") {
		runtime.HTTPPort = "9080"
	}

	runtime.WorkerGroups[schema.DEFAULT_WORKER_GROUP] = true
	runtime.WorkerQueues = make(map[string]*ContractQueue[activity.PipelineActivity], len(runtime.WorkerGroups))
	runtime.Activity = make(map[string]*activity.RuntimeActivity, len(runtime.WorkerGroups))
	for group := range runtime.WorkerGroups {
		runtime.WorkerQueues[group] = &ContractQueue[activity.PipelineActivity]{Queue: queue.Blocked(queue.NewQueue[activity.PipelineActivity]())}

		notifications := make(chan schema.PipelineStatus)

		runtime.Activity[group] = activity.NewRuntimeActivity(group, TIMEOUT_ACTIVITY, notifications)

		go func() {
			for {
				status := <-notifications

				runtime.NotifyQueue.Push(status)

				runtime.Status <- []string{fmt.Sprintf("[%s|%s: %s] %s", status.WorkerGroup, status.ActivityID, status.Pipeline.Name, status.Status)}

				switch status.Status {
				case schema.STATUS_RUNNING:
					go func() {
						reader, err := status.Logs.Reader()
						if err != nil {
							fmt.Printf("### [%s|%s: %s] reading logs failed - %s\n", status.WorkerGroup, status.ActivityID, status.Pipeline.Name, err)
							return
						}

						defer reader.Close()

						err = FilterPipeline(reader, os.Stdout, fmt.Sprintf("### [%s|%s] > ", status.WorkerGroup, status.ActivityID))
						if err != nil {
							fmt.Printf("### [%s|%s: %s] reading logs failed - %s\n", status.WorkerGroup, status.ActivityID, status.Pipeline.Name, err)
						}
					}()

				case schema.STATUS_SUCCESS, schema.STATUS_FAILED, schema.STATUS_TIMEOUT:
					runtime.LogQueueStatus()
				}
			}
		}()
	}

	return &runtime
}

func (runtime *Runtime) LogQueueStatus() {
	total := uint(0)
	lines := make([]string, 1+len(runtime.WorkerQueues))
	i := 0

	for group, queue := range runtime.WorkerQueues {
		i += 1
		count := queue.Count()
		total += count
		lines[i] = fmt.Sprintf("<status>   -> %s: %v", group, count)
	}

	lines[0] = fmt.Sprintf("<status> %v pipelines enqueued in:", total)

	runtime.Status <- lines
}

func (runtime *Runtime) LogStatus() {
	for {
		lines := <-runtime.Status

		for _, line := range lines {
			runtime.Log.Println(line)
		}
	}
}

var outputRegex *regexp.Regexp = regexp.MustCompile(`^\[(setup|step:\d+):>\]`)

func FilterPipeline(r io.Reader, w io.Writer, prefix string) error {
	return filter.LineFilter(r, w, func(line string) string {
		if outputRegex.MatchString(line) {
			return ""
		}
		return prefix + line
	})
}
