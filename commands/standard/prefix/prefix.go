package prefix

import (
	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/bot/prefix"
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/dispatch"
)

var Command = &dispatch.Command{
	Command:     "prefix",
	Category:    command.CategoryGeneral,
	Description: "Views the bot prefix",
	Args: []*dispatch.Arg{
		{Name: "Prefix", Type: dispatch.String},
	},
	Run: (func(data *dispatch.Data) error {
		prefix := prefix.GuildPrefix(data.Guild.ID)
		functions.SendBasicMessage(data.Channel.ID, "This servers prefix is `"+prefix+"`")

		return nil
	}),
}
