package main

import (
	"github.com/reeveci/reeve-lib/plugin"
	"github.com/reeveci/reeve-lib/queue"
	"github.com/reeveci/reeve-lib/schema"
	"github.com/reeveci/reeve/reeve-server/runtime"
)

func HandleMessageQueues(runtime *runtime.Runtime) {
	for name, queue := range runtime.MessageQueues {
		go HandleMessageQueue(runtime, queue, runtime.PluginProvider.MessagePlugins[name])
	}

	for {
		message := runtime.MessageQueue.Pop()

		switch message.Target {
		case schema.BROADCAST_MESSAGE:
			for _, queue := range runtime.MessageQueues {
				queue.Push(message)
			}

		default:
			if queue, ok := runtime.MessageQueues[message.Target]; ok {
				queue.Push(message)
			}
		}
	}
}

func HandleMessageQueue(runtime *runtime.Runtime, queue queue.Queue[schema.FullMessage], plugin plugin.Plugin) {
	for {
		message := queue.Pop()

		err := plugin.Message(message.Source, message.Message)
		if err != nil {
			runtime.ErrorLog.Printf("sending message to plugin %s failed - %s\n", message.Target, err)
			continue
		}
	}
}
