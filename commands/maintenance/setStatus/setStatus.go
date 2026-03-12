package setstatus

import (
	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/dispatch"
	"github.com/bwmarrin/discordgo"
)

var Command = &dispatch.Command{
	Command:      "setstatus",
	Category:     command.CategoryOwner,
	Description:  "Changes the bot status",
	ArgsRequired: 1,
	Args: []*dispatch.Arg{
		{Name: "Status", Type: dispatch.String},
	},
	Run: command.OwnerCommand(func(data *dispatch.Data) error {
		status := data.ParsedArgs[0].Value.(string)
		functions.SetStatus(status)
		message := &discordgo.MessageSend{
			Content: "Status changed",
		}

		functions.SendMessage(data.Channel.ID, message)

		return nil
	}),
}
