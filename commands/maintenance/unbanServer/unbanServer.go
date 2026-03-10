package unbanserver

import (
	"context"
	"errors"

	"github.com/RhykerWells/Summit/bot/core/models"
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/dispatch"
)

var Command = &dispatch.Command{
	Command:      "unbanserver",
	Category:     command.CategoryOwner,
	Description:  "Removes the server ban from inviting the bot",
	ArgsRequired: 1,
	Args: []*dispatch.Arg{
		{Name: "GuildID", Type: dispatch.String},
	},
	Run: command.OwnerCommand(func(data *dispatch.Data) error {
		guildID := data.ParsedArgs[0].Value.(string)
		banned := command.IsGuildBanned(guildID)
		if !banned {
			return errors.New("That guild was not banned")
		}

		models.BannedGuilds(models.BannedGuildWhere.GuildID.EQ(guildID)).DeleteAll(context.Background(), common.PQ)
		common.Session.MessageReactionAdd(data.Channel.ID, data.Message.ID, "👍")

		return nil
	}),
}
