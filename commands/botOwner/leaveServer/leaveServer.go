package leaveserver

import (
	"fmt"

	"github.com/RhykerWells/Summit/commands/util"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/common/dcommand"
)

var Command = &dcommand.SummitCommand{
	Command:      "leaveserver",
	Category:     dcommand.CategoryOwner,
	Description:  "Forces the bot to leave a given server",
	ArgsRequired: 1,
	Args: []*dcommand.Arg{
		{Name: "GuildID", Type: dcommand.String},
	},
	Run: util.OwnerCommand(func(data *dcommand.Data) error {
		err := common.Session.GuildLeave(data.ParsedArgs[0].String())
		if err != nil {
			return fmt.Errorf("Was unable to leave the guild: %s", err.Error())
		}

		common.Session.MessageReactionAdd(data.ChannelID, data.Message.ID, "👍")

		return nil
	}),
}
