package util

import (
	"context"
	"errors"

	"github.com/RhykerWells/Summit/bot/core/models"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/common/dcommand"
	"github.com/bwmarrin/discordgo"
)

func OwnerCommand(inner dcommand.Run) dcommand.Run {
	return func(data *dcommand.Data) error {
		if data.Author.ID == common.ConfigBotOwner {
			inner(data)
		} else {
			return errors.New("This is a bot-owner only command.")
		}

		return nil
	}
}

func AdminOrManageServerCommand(inner dcommand.Run) dcommand.Run {
	return func(data *dcommand.Data) error {
		perms, _ := data.Session.State.UserChannelPermissions(data.Author.ID, data.ChannelID)
		if perms&discordgo.PermissionAdministrator == 8 || perms&discordgo.PermissionManageServer == 32 {
			inner(data)
		} else {
			return errors.New("You need `Administrator` or `ManageServer` permissions to use this command.")
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
