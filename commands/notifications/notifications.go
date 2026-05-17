package notifications

import (
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/Summit/common"
)

//go:generate sqlboiler --no-hooks psql

func RegisterPlugin() {
	common.RegisterPlugin(&Plugin{})

	common.InitSchema("Notifications", GuildNotificationSchema...)
}

type Plugin struct{}

func (p *Plugin) PluginInfo() *common.PluginInfo {
	return &common.PluginInfo{
		Name:     "Notifications",
		Category: &command.CategoryMisc,
	}
}

func (p *Plugin) InitWeb() {
	initWeb()
}

func (p *Plugin) InitBot() {
	initEvents()
}
