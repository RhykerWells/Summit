package command

import (
	"github.com/RhykerWells/Summit/bot/prefix"
	"github.com/RhykerWells/dispatch"
	"github.com/bwmarrin/discordgo"
)

var CommandHandler *dispatch.CommandHandler

func InitCommandHandler(session *discordgo.Session) {
	CommandHandler = dispatch.NewCommandHandler()

	CommandHandler.SetPrefixFunc(prefix.GuildPrefix)
}
