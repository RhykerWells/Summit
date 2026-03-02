package setstatus

import (
	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/commands/util"
	"github.com/RhykerWells/Summit/common/dcommand"
	"github.com/bwmarrin/discordgo"
)

var Command = &dcommand.SummitCommand{
	Command:      "setstatus",
	Category:     dcommand.CategoryOwner,
	Description:  "Changes the bot status",
	ArgsRequired: 1,
	Args: []*dcommand.Arg{
		{Name: "Status", Type: dcommand.String},
	},
	Run: util.OwnerCommand(func(data *dcommand.Data) error {
		functions.SetStatus(data.ParsedArgs[0].String())
		message := &discordgo.MessageSend{
			Content: "Status changed",
		}

		functions.SendMessage(data.ChannelID, message)

		return nil
	}),
}
