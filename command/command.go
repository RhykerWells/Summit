package command

import (
	"github.com/RhykerWells/Summit/bot/prefix"
	"github.com/RhykerWells/dispatch"
	"github.com/bwmarrin/discordgo"
)

var CommandHandler *dispatch.CommandHandler

func InitCommandHandler(session *discordgo.Session) {
	CommandHandler = dispatch.NewCommandHandler()
	session.AddHandler(CommandHandler.HandleMessageCreate)

	CommandHandler.SetPrefixFunc(prefix.GuildPrefix)
}

func RegisterCommands(commands ...*dispatch.Command) {
	CommandHandler.RegisterCommands(commands...)
}
