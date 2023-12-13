package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/google/uuid"
	"github.com/reeveci/reeve-lib/conditions"
	"github.com/reeveci/reeve-lib/exe"
	"github.com/reeveci/reeve-lib/filter"
	"github.com/reeveci/reeve-lib/logs"
	"github.com/reeveci/reeve-lib/schema"
)

var stepDefaultConditions = map[string]schema.Condition{
	"status": {
		Include: []string{"success"},
	},
}

type Runtime struct {
	Pipeline schema.Pipeline

	Log, ErrorLog logs.LogWriter

	APIPort       string
	RuntimeEnv    string
	DockerCommand string
	NoDescription bool

	hostname        string
	network         string
	workspaceVolume string
	ApiUrl          string

	VarLock sync.Mutex
	Vars    map[string]schema.Var

	filters []string

	networkCreated         bool
	networkAttached        bool
	workspaceVolumeCreated bool

	cancelLock sync.Mutex
	canceled   bool
}

func GetRuntime() (*Runtime, error) {
	runtime := Runtime{
		APIPort:       exe.GetEnvDef("REEVE_API_PORT", "9090"),
		RuntimeEnv:    exe.GetEnvDef("REEVE_RUNTIME_ENV", "host"),
		DockerCommand: exe.GetEnvDef("REEVE_DOCKER_COMMAND", "docker"),
		NoDescription: exe.GetBoolEnvDef("REEVE_NO_DESCRIPTION", false),

		network:         "reeve-" + uuid.NewString(),
		workspaceVolume: "reeve-" + uuid.NewString(),

		Vars: make(map[string]schema.Var),
	}

	var err error
	runtime.hostname, err = os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("error resolving hostname - %s", err)
	}

	runtime.ApiUrl = fmt.Sprintf("http://%s:%s", runtime.hostname, runtime.APIPort)

	return &runtime, nil
}

func (runtime *Runtime) Prepare() bool {
	errorLog := runtime.ErrorLog.Subsystem("prepare")

	runtime.filters = make([]string, 0, len(runtime.Pipeline.Env))
	for _, env := range runtime.Pipeline.Env {
		if env.Secret {
			runtime.filters = append(runtime.filters, env.Value)
		}
	}

	success, err := runtime.RunCommand(runtime.DockerCommand, []string{"network", "create", runtime.network}, nil, nil)
	if err != nil {
		errorLog.Printf("failed to create network - %s\n", err)
		return false
	}
	if !success {
		errorLog.Printf("failed to create network\n")
		return false
	}
	runtime.networkCreated = true

	if runtime.RuntimeEnv == "docker" {
		success, err := runtime.RunCommand(runtime.DockerCommand, []string{"network", "connect", runtime.network, runtime.hostname}, nil, nil)
		if err != nil {
			errorLog.Printf("failed to attach network - %s\n", err)
			return false
		}
		if !success {
			errorLog.Printf("failed to attach network\n")
			return false
		}
		runtime.networkAttached = true
	}

	success, err = runtime.RunCommand(runtime.DockerCommand, []string{"volume", "create", runtime.workspaceVolume}, nil, nil)
	if err != nil {
		errorLog.Printf("failed to create workspace volume - %s\n", err)
		return false
	}
	if !success {
		errorLog.Printf("failed to create workspace volume\n")
		return false
	}
	runtime.workspaceVolumeCreated = true

	return true
}

func (runtime *Runtime) Cleanup() {
	errorLog := runtime.ErrorLog.Subsystem("cleanup")

	if runtime.workspaceVolumeCreated {
		success, err := runtime.RunCommand(runtime.DockerCommand, []string{"volume", "rm", "-f", runtime.workspaceVolume}, nil, nil)
		if err != nil {
			errorLog.Printf("failed to remove workspace volume - %s\n", err)
		} else if !success {
			errorLog.Printf("failed to remove workspace volume\n")
		}
	}

	if runtime.networkAttached {
		success, err := runtime.RunCommand(runtime.DockerCommand, []string{"network", "disconnect", "-f", runtime.network, runtime.hostname}, nil, nil)
		if err != nil {
			errorLog.Printf("failed to detach network - %s\n", err)
		} else if !success {
			errorLog.Printf("failed to detach network\n")
		}
	}

	if runtime.networkCreated {
		success, err := runtime.RunCommand(runtime.DockerCommand, []string{"network", "rm", runtime.network}, nil, nil)
		if err != nil {
			errorLog.Printf("failed to remove network - %s\n", err)
		} else if !success {
			errorLog.Printf("failed to remove network\n")
		}
	}
}

func (runtime *Runtime) Cancel() {
	runtime.cancelLock.Lock()
	defer runtime.cancelLock.Unlock()

	runtime.canceled = true
}

func (runtime *Runtime) Run() bool {
	defer runtime.Cleanup()

	if !runtime.NoDescription && strings.TrimSpace(runtime.Pipeline.Description) != "" {
		description, err := glamour.Render(runtime.Pipeline.Description, "notty")
		if err != nil {
			runtime.ErrorLog.Subsystem("description").Printf("failed to render description - %s\n", err)
		} else {
			runtime.Log.Subsystem("description").Println(description)
		}
	}

	runtime.Log.Subsystem("prepare").Printf("setting up pipeline %s\n", runtime.Pipeline.Name)
	if !runtime.Prepare() {
		runtime.ErrorLog.Subsystem("prepare").Println("failed to set up pipeline - exiting")
		return false
	}

	setupLog := runtime.Log.Subsystem("setup")
	setupErrorLog := runtime.ErrorLog.Subsystem("setup")
	success := runtime.RunTask(runtime.Pipeline.Setup.RunConfig, setupLog, setupErrorLog)
	if success {
		setupLog.Subsystem("success").Println("setup done")
	} else {
		setupLog.Subsystem("failure").Println("setup failed")
		return false
	}

	pipelineSuccess := true
	if runtime.Pipeline.Facts == nil {
		runtime.Pipeline.Facts = make(map[string]schema.Fact)
	}

	stageStatuses := make(map[string]string)

	stepCount := len(runtime.Pipeline.Steps)
	for i, step := range runtime.Pipeline.Steps {
		stepNumber := i + 1
		stepLog := runtime.Log.Subsystem("step").Subsystem(strconv.Itoa(stepNumber))
		stepErrorLog := runtime.ErrorLog.Subsystem("step").Subsystem(strconv.Itoa(stepNumber))

		if !runtime.CheckHealth() {
			runtime.ErrorLog.Subsystem("signal").Println("pipeline execution canceled - exiting")
			return false
		}

		stage := step.Stage
		if stage == "" {
			stage = schema.DEFAULT_STAGE
		}

		if _, ok := stageStatuses[stage]; !ok {
			stageStatuses[stage] = "success"
		}
		runtime.Pipeline.Facts["status"] = []string{stageStatuses[stage]}

		runtime.VarLock.Lock()

		conditions.ApplyDefaults(&step.When, stepDefaultConditions)

		success, err := conditions.Check(runtime.Pipeline.Facts, step.When, runtime.Pipeline.Env, runtime.Vars)

		runtime.VarLock.Unlock()

		if err != nil {
			stepErrorLog.Printf("checking conditions failed - %s\n", err)
		}
		if !success || err != nil {
			if stage == schema.DEFAULT_STAGE {
				stepLog.Subsystem("skip").Printf("skipping step %s (%v/%v)\n", step.Name, stepNumber, stepCount)
			} else {
				stepLog.Subsystem("skip").Printf("skipping step %s [stage %s] (%v/%v)\n", step.Name, stage, stepNumber, stepCount)
			}
			continue
		}

		if stage == schema.DEFAULT_STAGE {
			stepLog.Printf("running step %s (%v/%v)\n", step.Name, stepNumber, stepCount)
		} else {
			stepLog.Printf("running step %s [stage %s] (%v/%v)\n", step.Name, stage, stepNumber, stepCount)
		}

		success = runtime.RunTask(step.RunConfig, stepLog, stepErrorLog)
		if success {
			stepLog.Subsystem("success").Printf("step %s done\n", step.Name)
		} else if step.IgnoreFailure {
			stepLog.Subsystem("success").Printf("step %s done - ignoring failure\n", step.Name)
		} else {
			stepLog.Subsystem("failure").Printf("step %s failed\n", step.Name)
			stageStatuses[stage] = "failure"
			pipelineSuccess = false
		}
	}

	if pipelineSuccess {
		runtime.Log.Subsystem("success").Println("pipeline finished successfully")
	} else {
		runtime.Log.Subsystem("failure").Println("pipeline finished unsuccessfully")
	}
	return pipelineSuccess
}

func (runtime *Runtime) RunTask(config schema.RunConfig, log, errorLog logs.LogWriter) bool {
	image := config.Task
	var trusted bool

	if strings.HasPrefix(image, "@") {
		var found bool
		for domain, imagePrefix := range runtime.Pipeline.TaskDomains {
			prefix := fmt.Sprintf("@%s/", domain)
			if domain == "" || !strings.HasPrefix(image, prefix) {
				continue
			}

			found = true
			image = imagePrefix + strings.TrimPrefix(image, prefix)
			for _, trustedDomain := range runtime.Pipeline.TrustedDomains {
				if domain == trustedDomain {
					trusted = true
					break
				}
			}
			break
		}

		if !found {
			errorLog.Printf("failed to run task - unknown task domain %s\n", strings.SplitN(image[1:], "/", 2)[0])
			return false
		}
	}

	if image == "" {
		errorLog.Printf("failed to run task - task image missing\n")
		return false
	}

	if !trusted {
		imageWithoutVersion := image
		imageName := imageWithoutVersion[strings.LastIndex(imageWithoutVersion, "/")+1:]
		if index := strings.LastIndex(imageName, ":"); index >= 0 {
			imageWithoutVersion = strings.TrimSuffix(imageWithoutVersion, imageName[index:])
		}
		for _, trustedTask := range runtime.Pipeline.TrustedTasks {
			if trustedTask != "" && imageWithoutVersion == trustedTask {
				trusted = true
				break
			}
		}
	}

	args := []string{
		"run",
		"--rm",
		"-i",
		"-v", fmt.Sprintf("%s:/reeve", runtime.workspaceVolume),
		"--network", runtime.network,
	}

	runtime.VarLock.Lock()
	resolvedConfig, missingEnv, _, err := config.Resolve(runtime.Pipeline.Env, runtime.Vars)
	runtime.VarLock.Unlock()

	if err != nil {
		errorLog.Printf("failed to run task - resolving params failed - %s\n", err)
		return false
	}
	if len(missingEnv) > 0 {
		errorLog.Printf("failed to run task - missing environment variables %s\n", strings.Join(missingEnv, ", "))
		return false
	}

	paramKeys := make([]string, 0, len(resolvedConfig.Params))
	for key, value := range resolvedConfig.Params {
		if strings.Contains(key, "=") {
			errorLog.Printf("failed to run task - invalid token '=' in param name '%s'\n", key)
			return false
		}
		paramKeys = append(paramKeys, key)
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}
	paramValue, err := json.Marshal(paramKeys)
	if err != nil {
		errorLog.Printf("failed to run task - error encoding task param keys - %s\n", err)
	}
	args = append(args, "-e", fmt.Sprintf("REEVE_PARAMS=%s", string(paramValue)))
	args = append(args, "-e", fmt.Sprintf("REEVE_API=%s", runtime.ApiUrl))

	if trusted {
		args = append(args, "-v", "/var/run/docker.sock:/var/run/docker.sock")
	}

	if len(resolvedConfig.Directory) > 0 {
		args = append(args,
			"-v", fmt.Sprintf("/%s:/reeve/mount:rw", strings.TrimPrefix(resolvedConfig.Directory, "/")),
			"-w", "/reeve/mount",
		)
	}

	if resolvedConfig.User != "" {
		args = append(args, "-u", resolvedConfig.User)
	}

	var input io.Reader
	if len(resolvedConfig.Input) > 0 {
		input = strings.NewReader(resolvedConfig.Input)
	}

	args = append(args, "--name", "reeve-"+uuid.NewString(), image)

	if len(config.Command) > 0 {
		args = append(args, config.Command...)
	}

	success, err := runtime.RunCommand(runtime.DockerCommand, args, input, log.Subsystem(">"))
	if err != nil {
		errorLog.Printf("failed to run task - %s\n", err)
		return false
	}

	return success
}

func (runtime *Runtime) CheckHealth() bool {
	runtime.cancelLock.Lock()
	defer runtime.cancelLock.Unlock()

	return !runtime.canceled
}

func (runtime *Runtime) RunCommand(command string, args []string, stdin io.Reader, log logs.LogWriter) (success bool, err error) {
	cmd := exec.Command(command, args...)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	var outReader io.Reader
	if log != nil {
		outReader, err = cmd.StdoutPipe()
		if err != nil {
			return
		}
		cmd.Stderr = cmd.Stdout
	}
	err = cmd.Start()
	if err != nil {
		return
	}
	if outReader != nil {
		err := FilterSensitive(outReader, log, runtime.filters)
		if err != nil {
			panic(err)
		}
	}
	err = cmd.Wait()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func FilterSensitive(r io.Reader, w io.Writer, filters []string) error {
	return filter.LineFilter(r, w, func(line string) string {
		for _, f := range filters {
			line = strings.ReplaceAll(line, f, "*******")
		}
		return line
	})
}
