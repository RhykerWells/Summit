package command

import (
	"context"
	"errors"

	"github.com/RhykerWells/Summit/bot/core/models"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/dispatch"
)

func OwnerCommand(inner dispatch.Run) dispatch.Run {
	return func(data *dispatch.Data) error {
		if data.Author.ID == common.ConfigBotOwner {
			inner(data)
		} else {
			return errors.New("This is a bot-owner only command.")
		}

		return nil
	}
}

// IsGuildBanned returns a boolean of whether the guild is banned or not
func IsGuildBanned(guildID string) bool {
	exists, err := models.BannedGuilds(models.BannedGuildWhere.GuildID.EQ(guildID)).Exists(context.Background(), common.PQ)
	if err != nil {
		return false
	}

	return exists
}

func HasPerms(guildID, channelID, userID string, perm int64) bool {
	perms, err := common.Session.State.UserChannelPermissions(userID, channelID)
	if err != nil {
		return false
	}

	hasPerm := perms&perm != 0
	return hasPerm
}
