package moderation

import (
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/Summit/common"
)

//go:generate sqlboiler --no-hooks psql

var (
	MessageStore *MessageCache
)

func RegisterPlugin() {
	common.RegisterPlugin(&Plugin{})

	common.InitSchema("Moderation", GuildModerationSchema...)
	MessageStore = NewMessageCache(100)
}

type Plugin struct{}

func (p *Plugin) PluginInfo() *common.PluginInfo {
	return &common.PluginInfo{
		Name:     "Moderation",
		Category: &command.CategoryModeration,
	}
}

func (p *Plugin) InitCommands() {
	command.RegisterCommands(moderationCommands...)
	command.RegisterCommands(moderationHelpers...)
}

func (p *Plugin) InitWeb() {
	initWeb()
}

func (p *Plugin) InitBot() {
	initEvents()
	scheduleAllPendingUnmutes()
	scheduleAllPendingUnbans()
	refreshAllMuteSettings()
}
