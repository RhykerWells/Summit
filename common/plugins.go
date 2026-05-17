package common

import (
	"github.com/RhykerWells/dispatch"
	"github.com/sirupsen/logrus"
)

var (
	Plugins []Plugin
)

// Plugin represents a plugin, all plugins needs to implement this at a bare minimum
// and expose metadata about itself.
type Plugin interface {
	PluginInfo() *PluginInfo
}

// PluginInfo contains metadata about a plugin.
type PluginInfo struct {
	Name     string
	Category *dispatch.CommandCategory
}

// PluginWithCommands can register commands and message handlers during bot startup.
type PluginWithCommands interface {
	Plugin
	InitCommands()
}

// PluginWithWeb can initialise web routes and pages for a plugin.
type PluginWithWeb interface {
	Plugin
	InitWeb()
}

// PluginWithBotInit can initialise bot-specific setup after the discord session exists.
type PluginWithBotInit interface {
	Plugin
	InitBot()
}

func RegisterPlugin(plugin Plugin) {
	Plugins = append(Plugins, plugin)
	logrus.Infof("Registered plugin: %s", plugin.PluginInfo().Name)
}

func InitPluginCommands() {
	for _, plugin := range Plugins {
		if cast, ok := plugin.(PluginWithCommands); ok {
			cast.InitCommands()
		}
	}
}

func InitWebPlugins() {
	for _, plugin := range Plugins {
		if cast, ok := plugin.(PluginWithWeb); ok {
			cast.InitWeb()
		}
	}
}

func InitBotPlugins() {
	for _, plugin := range Plugins {
		if cast, ok := plugin.(PluginWithBotInit); ok {
			cast.InitBot()
		}
	}
}
