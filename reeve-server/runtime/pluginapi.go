package runtime

import (
	"github.com/reeveci/reeve-lib/schema"
)

func NewPluginAPI(plugin string, runtime *Runtime) *PluginAPI {
	return &PluginAPI{Plugin: plugin, Runtime: runtime}
}

type PluginAPI struct {
	Plugin  string
	Runtime *Runtime
}

func (api *PluginAPI) NotifyMessages(messages []schema.Message) error {
	for _, message := range messages {
		api.Runtime.MessageQueue.Push(schema.FullMessage{Message: message, Source: api.Plugin})
	}

	return nil
}

func (api *PluginAPI) NotifyTriggers(triggers []schema.Trigger) error {
	for _, trigger := range triggers {
		api.Runtime.TriggerQueue.Push(trigger)
	}

	return nil
}

func (api *PluginAPI) Close() error {
	return nil
}
