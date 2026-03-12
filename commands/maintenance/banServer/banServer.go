package banserver

import (
	"context"
	"errors"

	"github.com/RhykerWells/Summit/bot/core/models"
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/dispatch"
	"github.com/aarondl/sqlboiler/v4/boil"
)

var Command = &dispatch.Command{
	Command:      "ban",
	Category:     command.CategoryOwner,
	Description:  "Bans a server from inviting the bot",
	ArgsRequired: 1,
	Args: []*dispatch.Arg{
		{Name: "GuildID", Type: dispatch.String},
	},
	Run: command.OwnerCommand(func(data *dispatch.Data) error {
		guildID := data.ParsedArgs[0].Value.(string)
		banned := command.IsGuildBanned(guildID)
		if banned {
			return errors.New("This guild is already banned")
		} else {
			guild := models.BannedGuild{
				GuildID: guildID,
			}
			guild.Insert(context.Background(), common.PQ, boil.Infer())
		}

		return nil
	}),
}
