package core

//go:generate sqlboiler --no-hooks psql

import (
	eventsv2 "github.com/RhykerWells/Summit/bot/eventsV2"
)

// Init registers the required guild join & leave functions as well as initialises the web plugin
func Init() {
	eventsv2.AddHandler(handleGuildJoin, eventsv2.EventGuildCreate)
	eventsv2.AddHandler(handleGuildDelete, eventsv2.EventGuildDelete)

	initWeb()
}

func handleGuildJoin(data *eventsv2.EventData) error {
	g := data.GuildCreate()

	config := GetConfig(g.ID)
	SaveConfig(config)

	return nil
}

func handleGuildDelete(data *eventsv2.EventData) error {
	g := data.GuildDelete()

	config := GetConfig(g.ID)
	DeleteConfig(config)

	return nil
}
