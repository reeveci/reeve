package legacy

import (
	"github.com/reeveci/reeve-lib/conditions"
	"github.com/reeveci/reeve-lib/schema"
	"github.com/reeveci/reeve-lib/vars"
	"github.com/reeveci/reeve/server/legacy/runtime"
)

var pipelineDefaultConditions = map[string]schema.Condition{
	"workerGroup": {
		Include: []string{schema.DEFAULT_WORKER_GROUP},
	},
}

type paramPipeline struct {
	Params       vars.PipelineEnvBundle
	WorkerGroups []string
	schema.Pipeline
}

func HandleTriggerQueue(runtime *runtime.Runtime) {
	for {
		trigger := runtime.TriggerQueue.Pop()

		pipelines := discoverPipelines(runtime, trigger)
		if len(pipelines) == 0 {
			continue
		}

		pipelineEnvMap := make(map[string]bool)
		for _, pipeline := range pipelines {
			for _, key := range pipeline.Params.PipelineEnv {
				pipelineEnvMap[key] = true
			}
		}
		resolvedPipelineEnv := resolveEnv(runtime, pipelineEnvMap)

		remainingPipelines := make([]*paramPipeline, 0, len(pipelines))

	L:
		for _, pipeline := range pipelines {
			pipelineEnv, err := vars.MergeEnv(pipeline.Params.PipelineEnv, pipeline.Env, resolvedPipelineEnv)
			if err != nil {
				runtime.ErrorLog.Printf("error analyzing pipeline %s - %s\n", pipeline.Name, err)
				continue
			}

			conditions.ApplyDefaults(&pipeline.When, pipelineDefaultConditions)

			pipelineWhen := make(map[string]schema.Condition, len(pipeline.When))
			workerWhen := make(map[string]schema.Condition, 1)
			for key, value := range pipeline.When {
				switch key {
				case "workerGroup":
					workerWhen[key] = value
				default:
					pipelineWhen[key] = value
				}
			}

			ok, err := conditions.Check(pipeline.Facts, pipelineWhen, pipelineEnv, nil)
			if err != nil {
				runtime.ErrorLog.Printf("checking conditions for pipeline %s failed - %s\n", pipeline.Name, err)
				continue
			}
			if !ok {
				continue
			}

			pipeline.WorkerGroups = make([]string, 0, len(runtime.WorkerGroups))
			for group := range runtime.WorkerGroups {
				ok, err = conditions.Check(map[string]schema.Fact{
					"workerGroup": {group},
				}, workerWhen, pipelineEnv, nil)
				if err != nil {
					runtime.ErrorLog.Printf("checking conditions for pipeline %s failed - %s\n", pipeline.Name, err)
					continue L
				}
				if ok {
					pipeline.WorkerGroups = append(pipeline.WorkerGroups, group)
				}
			}

			if len(pipeline.WorkerGroups) > 0 {
				remainingPipelines = append(remainingPipelines, pipeline)
			}
		}
		if len(remainingPipelines) == 0 {
			continue
		}

		remainingEnvMap := make(map[string]bool)
		for _, pipeline := range remainingPipelines {
			for _, key := range pipeline.Params.RemainingEnv {
				remainingEnvMap[key] = true
			}
		}
		resolvedRemainingEnv := resolveEnv(runtime, remainingEnvMap)

		for _, pipeline := range remainingPipelines {
			env, err := vars.MergeEnv(pipeline.Params.Env, pipeline.Env, resolvedPipelineEnv, resolvedRemainingEnv)
			if err != nil {
				runtime.ErrorLog.Printf("error running pipeline %s - %s\n", pipeline.Name, err)
				continue
			}
			pipeline.Env = env

			for _, group := range pipeline.WorkerGroups {
				workerPipeline := pipeline.Pipeline
				workerPipeline.Facts = make(map[string]schema.Fact, len(pipeline.Facts))
				for key, value := range pipeline.Facts {
					workerPipeline.Facts[key] = value
				}
				workerPipeline.Facts["workerGroup"] = schema.Fact{group}

				activity := runtime.Activity[group].RegisterPipeline(workerPipeline)
				runtime.WorkerQueues[group].Push(activity)
			}
		}

		runtime.LogQueueStatus()
	}
}

func discoverPipelines(runtime *runtime.Runtime, trigger schema.Trigger) (result []*paramPipeline) {
	pluginCount := len(runtime.PluginProvider.DiscoverPlugins)

	if pluginCount > 0 {
		channel := make(chan []paramPipeline, pluginCount)

		for k, v := range runtime.PluginProvider.DiscoverPlugins {
			pluginName := k
			plugin := v

			go func() {
				pipelines, err := plugin.Discover(trigger)
				if err != nil {
					runtime.ErrorLog.Printf("discovering pipelines with plugin %s failed - %s\n", pluginName, err)
					channel <- nil
					return
				}

				toBeRun := make([]paramPipeline, 0, len(pipelines))
				for _, pipeline := range pipelines {
					if pipeline.Name == "" || len(pipeline.Steps) == 0 {
						runtime.ErrorLog.Printf("skipping empty pipeline %s\n", pipeline.Name)
						continue
					}

					toBeRun = append(toBeRun, paramPipeline{
						Params:   vars.FindAllEnv(pipeline),
						Pipeline: pipeline,
					})
				}

				channel <- toBeRun
			}()
		}

		for i := 0; i < pluginCount; i++ {
			discovered := <-channel

			for _, p := range discovered {
				pipeline := p
				result = append(result, &pipeline)
			}
		}
	}

	return
}

func resolveEnv(runtime *runtime.Runtime, envMap map[string]bool) (result map[string]schema.Env) {
	pluginCount := len(runtime.PluginProvider.ResolvePlugins)

	env := make([]string, 0, len(envMap))
	for key, ok := range envMap {
		if key != "" && ok {
			env = append(env, key)
		}
	}

	result = make(map[string]schema.Env)

	if len(env) > 0 && pluginCount > 0 {
		channel := make(chan map[string]schema.Env, pluginCount)

		for k, v := range runtime.PluginProvider.ResolvePlugins {
			pluginName := k
			plugin := v

			go func() {
				env, err := plugin.Resolve(env)
				if err != nil {
					runtime.ErrorLog.Printf("resolving environment variables with plugin %s failed - %s\n", pluginName, err)
					channel <- nil
					return
				}

				channel <- env
			}()
		}

		for i := 0; i < pluginCount; i++ {
			resolved := <-channel

			for key, value := range resolved {
				if key != "" {
					if existing, ok := result[key]; !ok || value.Priority < existing.Priority {
						result[key] = value
					}
				}
			}
		}
	}

	return
}
