package main

import (
	"github.com/RhykerWells/Summit/economy"
	"github.com/RhykerWells/Summit/moderation"
	"github.com/RhykerWells/Summit/notifications"
)

func initPlugins() {
	moderation.RegisterPlugin()
	economy.RegisterPlugin()
	notifications.RegisterPlugin()
}
