package moderation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/moderation/models"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/bwmarrin/discordgo"
)

var (
	errNotMuted  = errors.New("user not muted")
	errNotBanned = errors.New("user not banned")
	errNotMember = errors.New("user not a member")
)

// muteUser automatically applies a mute role to the target user and removes any roles set to be removed
// within the configuration and saves the mute to the database
func muteUser(config *Config, targetID string, duration *time.Duration) error {
	err := functions.AddRole(config.GuildID, targetID, config.MuteRole)
	if err != nil {
		return err
	}

	rolesRemoved := []string{}
	member, _ := functions.GetMember(config.GuildID, targetID)
	if len(config.MuteUpdateRoles) > 0 {
		roleSet := make(map[string]struct{}, len(config.MuteUpdateRoles))
		for _, role := range config.MuteUpdateRoles {
			roleSet[role] = struct{}{}
		}

		for _, userRole := range member.Roles {
			if _, exists := roleSet[userRole]; exists {
				err := functions.RemoveRole(config.GuildID, targetID, userRole)
				if err != nil {
					continue
				}
				rolesRemoved = append(rolesRemoved, userRole)
			}
		}
	}

	var unmuteTime time.Time
	if duration != nil {
		unmuteTime = time.Now().Add(*duration)
	}

	muteEntry := models.ModerationMute{
		GuildID:  config.GuildID,
		UserID:   targetID,
		Roles:    rolesRemoved,
		UnmuteAt: unmuteTime,
	}

	muteEntry.Upsert(context.Background(), common.PQ, true, []string{models.ModerationMuteColumns.GuildID, models.ModerationMuteColumns.UserID}, boil.Whitelist(models.ModerationMuteColumns.UnmuteAt), boil.Infer())

	if duration != nil {
		scheduleUnmute(config, targetID, unmuteTime)
	}

	return nil
}

// unmuteUser automatically removes the mute role from the target user and restores any roles set to be removed
// within the configuration and saves the mute to the database
func unmuteUser(config *Config, authorID, targetID string) error {
	mutedUser, err := models.ModerationMutes(models.ModerationMuteWhere.GuildID.EQ(config.GuildID), models.ModerationMuteWhere.UserID.EQ(targetID)).One(context.Background(), common.PQ)
	if err != nil {
		return errNotMuted
	}

	targetUser, err := functions.GetUser(targetID)
	if err != nil {
		if authorID == common.Bot.ID {
			mutedUser.Delete(context.Background(), common.PQ)
		}
		return errNotMember
	}

	for _, roleID := range mutedUser.Roles {
		functions.AddRole(config.GuildID, targetID, roleID)
	}
	functions.RemoveRole(config.GuildID, targetID, config.MuteRole)

	mutedUser.Delete(context.Background(), common.PQ)

	if authorID == common.Bot.ID {
		botUser, _ := functions.GetUser(common.Bot.ID)
		createCase(config, botUser, targetUser, logUnmute, nil, "Automatic unmute", nil)
		muteEmbed := buildDMEmbed(config, targetUser, logMute, "Automatic unmute", nil)
		functions.SendDM(targetID, &discordgo.MessageSend{Embed: muteEmbed})
	}

	return nil
}

// RefreshMuteSettings ensures that the configured mute role has correct
// permissions applied across all channels in the guild. It
func refreshMuteSettings(config *Config) {
	channels, err := common.Session.GuildChannels(config.GuildID)
	if err != nil {
		return
	}

	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildCategory {
			continue
		}

		refreshMuteSettingsOnChannel(config, channel)
	}
}

// refreshMuteSettingsOnChannel refreshes the mute settings on a channel that is missing the permission
func refreshMuteSettingsOnChannel(config *Config, channel *discordgo.Channel) {
	if hasPerms := command.HasPerms(config.GuildID, channel.ID, common.Bot.ID, discordgo.PermissionManageChannels); !hasPerms {
		return
	}

	if config.MuteRole == "" {
		return
	}

	if !config.MuteManageRole {
		return
	}

	var existingOverwrite *discordgo.PermissionOverwrite

	for _, v := range channel.PermissionOverwrites {
		if v.Type == discordgo.PermissionOverwriteTypeRole && v.ID == config.MuteRole {
			existingOverwrite = v
			break
		}
	}

	muteDenyPerms := int64(discordgo.PermissionSendMessages | discordgo.PermissionVoiceSpeak | discordgo.PermissionSendMessagesInThreads)
	channelPermsAllow := int64(0)
	channelPermsDeny := muteDenyPerms
	channelPermsChanged := true

	if existingOverwrite != nil {
		channelPermsAllow = existingOverwrite.Allow
		channelPermsDeny = existingOverwrite.Deny
		channelPermsChanged = false

		if channelPermsAllow&muteDenyPerms != 0 {
			// Remove any permissions that we're going to deny that are currently set to allow
			channelPermsAllow &= muteDenyPerms
			channelPermsChanged = false
		}

		if (channelPermsDeny & muteDenyPerms) != muteDenyPerms {
			// Add any permissions that we're going to deny that are currently not set
			channelPermsDeny |= muteDenyPerms
			channelPermsChanged = true
		}
	}

	if channelPermsChanged {
		common.Session.ChannelPermissionSet(channel.ID, config.MuteRole, discordgo.PermissionOverwriteTypeRole, channelPermsAllow, int64(muteDenyPerms))
	}
}

// scheduleUnmute triggers an unmute a target at unmuteTime
func scheduleUnmute(config *Config, targetID string, unmuteTime time.Time) {
	delay := time.Until(unmuteTime)
	if delay <= 0 {
		go unmuteUser(config, common.Bot.ID, targetID)
		return
	}

	go func() {
		time.Sleep(time.Until(unmuteTime))
		unmuteUser(config, common.Bot.ID, targetID)
	}()
}

// scheduleAllPendingUnmutes schedules unmutes for all users with pending unmutes
func scheduleAllPendingUnmutes() {
	mutes, err := models.ModerationMutes(models.ModerationMuteWhere.UnmuteAt.GT(time.Now())).All(context.Background(), common.PQ)
	if err != nil {
		return
	}

	for _, mute := range mutes {
		config := GetConfig(mute.GuildID)
		scheduleUnmute(config, mute.UserID, mute.UnmuteAt)
	}
}

// kickUser kicks the user from the given guild
func kickUser(config *Config, author, target *discordgo.User, reason string) error {
	auditLogReason := fmt.Sprintf("%s: %s", author.Username, reason)

	err := functions.GuildKickMember(config.GuildID, target.ID, auditLogReason)
	if err != nil {
		return err
	}

	return nil
}

// banUser bans the user from the current guild and adds an entry to the banned entry list
func banUser(config *Config, author, target *discordgo.User, reason string, duration *time.Duration) error {
	auditLogReason := fmt.Sprintf("%s: %s", author.Username, reason)

	err := functions.GuildBanMember(config.GuildID, target.ID, auditLogReason)
	if err != nil {
		return err
	}

	var unbanTime time.Time
	if duration != nil {
		unbanTime = time.Now().Add(*duration)
	}

	if duration != nil {
		banEntry := models.ModerationBan{
			GuildID: config.GuildID,
			UserID:  target.ID,
			UnbanAt: unbanTime,
		}
		banEntry.Upsert(context.Background(), common.PQ, true, []string{models.ModerationBanColumns.GuildID, models.ModerationBanColumns.UserID}, boil.Whitelist(models.ModerationBanColumns.UnbanAt), boil.Infer())
		scheduleUnban(config, target.ID, unbanTime)
	}
	return nil
}

// unbanUser removes a ban from a guild member and removes the user from the banned config
func unbanUser(config *Config, authorID, targetID string) error {
	bannedUser, _ := models.ModerationBans(models.ModerationBanWhere.GuildID.EQ(config.GuildID), models.ModerationBanWhere.UserID.EQ(targetID)).One(context.Background(), common.PQ)

	err := functions.GuildUnbanMember(config.GuildID, targetID)
	if err != nil {
		return errNotBanned
	}

	targetUser, _ := functions.GetUser(targetID)

	if bannedUser != nil {
		bannedUser.Delete(context.Background(), common.PQ)
	}

	if authorID == common.Bot.ID {
		botUser, _ := functions.GetUser(common.Bot.ID)
		createCase(config, botUser, targetUser, logUnban, nil, "Automatic unban", nil)
	}

	return nil
}

// scheduleUnban triggers an unban a target at unbanTime
func scheduleUnban(config *Config, targetID string, unbanTime time.Time) {
	delay := time.Until(unbanTime)
	if delay <= 0 {
		go unbanUser(config, common.Bot.ID, targetID)
		return
	}

	go func() {
		time.Sleep(time.Until(unbanTime))
		unbanUser(config, common.Bot.ID, targetID)
	}()
}

// scheduleAllPendingUnbans schedules unbans for all users with pending unbans
func scheduleAllPendingUnbans() {
	bannedUsers, err := models.ModerationBans(models.ModerationBanWhere.UnbanAt.GT(time.Now())).All(context.Background(), common.PQ)
	if err != nil {
		return
	}

	for _, bannedUser := range bannedUsers {
		config := GetConfig(bannedUser.GuildID)
		_, err := common.Session.GuildBan(bannedUser.GuildID, bannedUser.UserID)
		if err != nil {
			bannedUser.Delete(context.Background(), common.PQ)
			continue
		}

		scheduleUnban(config, bannedUser.UserID, bannedUser.UnbanAt)
	}
}

// refreshAllMuteSettings refreshes mute settings for all guilds on startup
func refreshAllMuteSettings() {
	for _, guild := range common.Session.State.Guilds {
		config := GetConfig(guild.ID)
		if config.MuteManageRole && config.MuteRole != "" {
			refreshMuteSettings(config)
		}
	}
}
