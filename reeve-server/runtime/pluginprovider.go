package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	goplugin "github.com/hashicorp/go-plugin"
	"github.com/reeveci/reeve-lib/plugin"
	"github.com/reeveci/reeve-lib/queue"
	"github.com/reeveci/reeve-lib/schema"
)

const SHARED_SETTING_PREFIX = "REEVE_SHARED_"

type CLIPlugin struct {
	plugin.Plugin
	CLIMethods map[string]string
}

type PluginProvider struct {
	Plugins map[string]plugin.Plugin

	MessagePlugins  map[string]plugin.Plugin
	DiscoverPlugins map[string]plugin.Plugin
	ResolvePlugins  map[string]plugin.Plugin
	NotifyPlugins   map[string]plugin.Plugin
	CLIPlugins      map[string]CLIPlugin
}

func (p *PluginProvider) Close() {
	var wg sync.WaitGroup
	wg.Add(len(p.Plugins))

	for _, v := range p.Plugins {
		plugin := v
		go func() {
			plugin.Unregister()
			wg.Done()
		}()
	}

	wg.Wait()

	goplugin.CleanupClients()
}

func (runtime *Runtime) LoadPlugins() error {
	plugin.RegisterSharedTypes()

	var pluginPaths []string
	pluginPaths, err := goplugin.Discover("*", runtime.PluginDirectory)
	if err != nil {
		return err
	}

	runtime.MessageQueues = make(map[string]queue.Queue[schema.FullMessage], len(pluginPaths))
	runtime.PluginProvider.Plugins = make(map[string]plugin.Plugin, len(pluginPaths))
	runtime.PluginProvider.MessagePlugins = make(map[string]plugin.Plugin, len(pluginPaths))
	runtime.PluginProvider.DiscoverPlugins = make(map[string]plugin.Plugin, len(pluginPaths))
	runtime.PluginProvider.ResolvePlugins = make(map[string]plugin.Plugin, len(pluginPaths))
	runtime.PluginProvider.NotifyPlugins = make(map[string]plugin.Plugin, len(pluginPaths))
	runtime.PluginProvider.CLIPlugins = make(map[string]CLIPlugin, len(pluginPaths))

	nameRegex := regexp.MustCompile("^[a-zA-Z0-9]+$")

	for _, path := range pluginPaths {
		client := goplugin.NewClient(&goplugin.ClientConfig{
			HandshakeConfig: plugin.Handshake,
			Plugins:         plugin.PluginMap,
			Cmd:             exec.Command(path),
			Managed:         true,
		})

		rpcClient, clientErr := client.Client()
		if clientErr != nil {
			goplugin.CleanupClients()
			return fmt.Errorf("error setting up plugin client for %s - %s", path, clientErr)
		}

		raw, pluginErr := rpcClient.Dispense("plugin")
		if pluginErr != nil {
			goplugin.CleanupClients()
			return fmt.Errorf("error dispensing plugin for %s - %s", path, pluginErr)
		}

		plugin := raw.(plugin.Plugin)

		name, err := plugin.Name()
		if err != nil {
			goplugin.CleanupClients()
			return fmt.Errorf("resolving plugin name failed for %s - %s", path, err)
		}

		if !nameRegex.MatchString(name) {
			goplugin.CleanupClients()
			return fmt.Errorf("plugin %s has invalid name, only [a-zA-Z0-9] are allowed - %s", name, err)
		}

		if _, exists := runtime.PluginProvider.Plugins[name]; exists {
			goplugin.CleanupClients()
			return fmt.Errorf("duplicate plugin %s", name)
		}

		runtime.PluginProvider.Plugins[name] = plugin
	}

	var wg sync.WaitGroup
	wg.Add(len(pluginPaths))
	errors := make(chan error, len(pluginPaths))
	var lock sync.Mutex

	for k, v := range runtime.PluginProvider.Plugins {
		name := k
		plugin := v

		go func() {
			defer wg.Done()

			settings := make(map[string]string)
			pluginPrefix := fmt.Sprintf("REEVE_PLUGIN_%s_", strings.ToUpper(name))
			for _, env := range os.Environ() {
				origKey := strings.Split(env, "=")[0]
				key := strings.ToUpper(origKey)
				if strings.HasPrefix(key, SHARED_SETTING_PREFIX) && len(key) > len(SHARED_SETTING_PREFIX)+1 {
					settingName := strings.TrimPrefix(key, SHARED_SETTING_PREFIX)
					if _, ok := settings[settingName]; !ok {
						settings[settingName] = os.Getenv(origKey)
					}
				}
				if strings.HasPrefix(key, pluginPrefix) && len(key) > len(pluginPrefix)+1 {
					settings[strings.TrimPrefix(key, pluginPrefix)] = os.Getenv(origKey)
				}
			}

			config, err := plugin.Register(settings, NewPluginAPI(name, runtime))
			if err != nil {
				errors <- fmt.Errorf("registering plugin %s failed - %s", name, err)
				return
			}

			lock.Lock()
			if config.Message {
				runtime.MessageQueues[name] = queue.Blocked(queue.NewQueue[schema.FullMessage]())
				runtime.PluginProvider.MessagePlugins[name] = plugin
			}
			if config.Discover {
				runtime.PluginProvider.DiscoverPlugins[name] = plugin
			}
			if config.Resolve {
				runtime.PluginProvider.ResolvePlugins[name] = plugin
			}
			if config.Notify {
				runtime.PluginProvider.NotifyPlugins[name] = plugin
			}
			if len(config.CLIMethods) > 0 {
				runtime.PluginProvider.CLIPlugins[name] = CLIPlugin{
					Plugin:     plugin,
					CLIMethods: config.CLIMethods,
				}
			}
			lock.Unlock()
		}()
	}
	wg.Wait()

	select {
	case err = <-errors:
		runtime.PluginProvider.Close()
		return err
	default:
	}

	return nil
}
