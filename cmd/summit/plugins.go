package main

import (
	"github.com/RhykerWells/Summit/commands/economy"
	"github.com/RhykerWells/Summit/commands/moderation"
	"github.com/RhykerWells/Summit/commands/notifications"
)

func initPlugins() {
	moderation.RegisterPlugin()
	economy.RegisterPlugin()
	notifications.RegisterPlugin()
}
