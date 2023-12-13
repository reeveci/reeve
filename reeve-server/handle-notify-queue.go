package main

import (
	"sync"

	"github.com/reeveci/reeve/reeve-server/runtime"
)

func HandleNotifyQueue(runtime *runtime.Runtime) {
	pluginCount := len(runtime.PluginProvider.NotifyPlugins)

	for {
		notification := runtime.NotifyQueue.Pop()

		if pluginCount > 0 {
			var wg sync.WaitGroup
			wg.Add(pluginCount)

			for k, v := range runtime.PluginProvider.NotifyPlugins {
				pluginName := k
				plugin := v

				go func() {
					defer wg.Done()

					err := plugin.Notify(notification)
					if err != nil {
						runtime.ErrorLog.Printf("sending notification to plugin %s failed - %s\n", pluginName, err)
						return
					}
				}()
			}

			wg.Wait()
		}
	}
}
