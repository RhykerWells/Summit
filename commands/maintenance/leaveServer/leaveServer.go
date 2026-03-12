package leaveserver

import (
	"fmt"

	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/dispatch"
)

var Command = &dispatch.Command{
	Command:      "leaveserver",
	Category:     command.CategoryOwner,
	Description:  "Forces the bot to leave a given server",
	ArgsRequired: 1,
	Args: []*dispatch.Arg{
		{Name: "GuildID", Type: dispatch.String},
	},
	Run: command.OwnerCommand(func(data *dispatch.Data) error {
		guildID := data.ParsedArgs[0].Value.(string)
		err := common.Session.GuildLeave(guildID)
		if err != nil {
			return fmt.Errorf("Was unable to leave the guild: %s", err.Error())
		}

		common.Session.MessageReactionAdd(data.Channel.ID, data.Message.ID, "👍")

		return nil
	}),
}
