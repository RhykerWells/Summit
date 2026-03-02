package commands

import (
	"github.com/RhykerWells/Summit/bot/functions"
	prfx "github.com/RhykerWells/Summit/bot/prefix"
	"github.com/RhykerWells/Summit/common/dcommand"
)

var prefixCmd = &dcommand.SummitCommand{
	Command:     "prefix",
	Category:    dcommand.CategoryGeneral,
	Description: "Views the bot prefix",
	Args: []*dcommand.Arg{
		{Name: "Prefix", Type: dcommand.String},
	},
	Run: (func(data *dcommand.Data) error {
		prefix := prfx.GuildPrefix(data.GuildID)
		functions.SendBasicMessage(data.ChannelID, "This servers prefix is `"+prefix+"`")

		return nil
	}),
}
