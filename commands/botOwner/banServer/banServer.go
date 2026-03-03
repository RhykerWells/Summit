package banserver

import (
	"context"
	"errors"

	"github.com/RhykerWells/Summit/bot/core/models"
	"github.com/RhykerWells/Summit/commands/util"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/common/dcommand"
	"github.com/aarondl/sqlboiler/v4/boil"
)

var Command = &dcommand.SummitCommand{
	Command:      "ban",
	Category:     dcommand.CategoryOwner,
	Description:  "Bans a server from inviting the bot",
	ArgsRequired: 1,
	Args: []*dcommand.Arg{
		{Name: "GuildID", Type: dcommand.String},
	},
	Run: util.OwnerCommand(func(data *dcommand.Data) error {
		banned := util.IsGuildBanned(data.ParsedArgs[0].String())
		if banned {
			return errors.New("This guild is already banned")
		} else {
			guild := models.BannedGuild{
				GuildID: data.ParsedArgs[0].String(),
			}
			guild.Insert(context.Background(), common.PQ, boil.Infer())
		}

		return nil
	}),
}
