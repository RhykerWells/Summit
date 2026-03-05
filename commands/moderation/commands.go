package moderation

import (
	"errors"
	"fmt"
	"time"

	"slices"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/common/dcommand"
	"github.com/bwmarrin/discordgo"
)

// moderationBase returns the config and a to denote the active enabled status.
func moderationBase(guildID string) (*Config, error) {
	config := GetConfig(guildID)

	if !config.ModerationEnabled {
		return nil, errors.New("The moderation system has not been enabled. Please enable it through the dashboard.")
	}

	if config.ModerationLogChannel == "" {
		return nil, errors.New("Please setup a modlog channel I can access through the dashboard before running this command.")
	}

	return config, nil
}

func moderationBasePermissionCheck(data *dcommand.Data, target *discordgo.User, discordPermissionRequired int64, additionalPermittanceRoles []string) error {
	perms, _ := common.Session.State.UserChannelPermissions(data.Author.ID, data.ChannelID)

	permsMet := false
	if perms&discordPermissionRequired != 0 {
		permsMet = true
	}

	if !permsMet && additionalPermittanceRoles != nil {
		permsMet = hasRequiredRole(data.GuildID, data.Author.ID, additionalPermittanceRoles)
	}

	if !permsMet {
		humanisedPerm, _ := dcommand.PermissionNames[discordPermissionRequired]
		return fmt.Errorf("This command requires either the `%s` permission or an additional role set up by admins through the dashboard.", humanisedPerm)
	}

	targetMember, _ := functions.GetMember(data.GuildID, target.ID)
	if targetMember != nil {
		author, _ := functions.GetMember(data.GuildID, data.Author.ID)
		ok := functions.IsMemberHigher(data.GuildID, author, targetMember)
		if !ok {
			return errors.New("You don't have the correct permissions to warn this user (target has higher or equal role).")
		}

		if targetMember.User.ID == author.User.ID {
			return errors.New("You can't run moderation commands on yourself.")
		}
	}

	return nil
}

// returns a boolean on whether the member has the current permissions to run the selected command
func hasRequiredRole(guildID, userID string, requiredRoles []string) bool {
	member, _ := functions.GetMember(guildID, userID)
	for _, role := range member.Roles {
		if slices.Contains(requiredRoles, role) {
			return true
		}
	}

	return false
}

// triggerDeletion returns the enabled status and time for the deleting the trigger
func triggerDeletion(config *Config) (bool, int64) {
	return config.ModerationTriggerDeletionEnabled, config.ModerationTriggerDeletionSeconds
}

// responseDeletion returns the enabled status and time for the deleting the response
func responseDeletion(config *Config) (bool, int64) {
	return config.ModerationResponseDeletionEnabled, config.ModerationResponseDeletionSeconds
}

// responseEmbed returns the fully-populated embed for responses
func responseEmbed(author, target *discordgo.User, action logAction) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    fmt.Sprintf("Case type: %s", action.CaseType),
			IconURL: author.AvatarURL("1024"),
		},
		Description: fmt.Sprintf("%s has successfully %s %s :thumbsup:", author.Mention(), action.Name, target.Mention()),
		Color:       0x242429,
	}
}

var moderationCommands = []*dcommand.SummitCommand{
	{
		Command:      "warn",
		Category:     dcommand.CategoryModeration,
		Aliases:      []string{""},
		Description:  "Warns a user for a specified reason",
		ArgsRequired: 2,
		Args: []*dcommand.Arg{
			{Name: "User", Type: dcommand.User},
			{Name: "Reason", Type: dcommand.String},
		},
		Run: func(data *dcommand.Data) error {
			config, err := moderationBase(data.GuildID)
			if err != nil {
				return err
			}

			target := data.ParsedArgs[0].User()
			err = moderationBasePermissionCheck(data, target, discordgo.PermissionModerateMembers, config.WarnRequiredRoles)
			if err != nil {
				return err
			}

			_, err = functions.GetMember(data.GuildID, target.ID)
			if err != nil {
				return errors.New("Member not found")
			}

			warnReason := data.ParsedArgs[1].String()

			err = createCase(config, data.Author, target, logWarn, data.ChannelID, warnReason)
			if err != nil {
				return fmt.Errorf("Something went wrong creating the case: %s", err.Error())
			}

			warnEmbed := buildDMEmbed(config, target, logWarn, warnReason)
			err = functions.SendDM(target.ID, &discordgo.MessageSend{Embed: warnEmbed})
			if err != nil {
				functions.SendBasicMessage(data.ChannelID, "Was not able to DM the user.")
			}

			ok, delay := triggerDeletion(config)
			if ok {
				functions.DeleteMessage(data.ChannelID, data.Message.ID, time.Duration(delay)*time.Second)
			}

			responseEmbed := responseEmbed(data.Author, target, logWarn)
			message, _ := functions.SendMessage(data.ChannelID, &discordgo.MessageSend{Embed: responseEmbed})
			ok, delay = responseDeletion(config)
			if ok {
				functions.DeleteMessage(data.ChannelID, message.ID, time.Duration(delay)*time.Second)
			}

			return nil
		},
	},
	{
		Command:          "mute",
		Category:         dcommand.CategoryModeration,
		Aliases:          []string{""},
		Description:      "Mutes a user for specified duration and reason",
		RequiredBotPerms: []int64{discordgo.PermissionAdministrator, discordgo.PermissionManageRoles},
		ArgsRequired:     3,
		Args: []*dcommand.Arg{
			{Name: "User", Type: dcommand.User},
			{Name: "Duration", Type: dcommand.Duration},
			{Name: "Reason", Type: dcommand.String},
		},
		ArgumentCombos: [][]int{
			{0, 1, 2},
			{0, 2, 1},
			{0, 1},
			{0, 2},
			{0},
		},
		Run: func(data *dcommand.Data) error {
			config, err := moderationBase(data.GuildID)
			if err != nil {
				return err
			}

			target := data.ParsedArgs[0].User()
			err = moderationBasePermissionCheck(data, target, discordgo.PermissionModerateMembers, config.MuteRequiredRoles)
			if err != nil {
				return err
			}

			_, err = functions.GetMember(data.GuildID, target.ID)
			if err != nil {
				return errors.New("Member not found")
			}

			var duration time.Duration
			var durationProvided bool

			if len(data.ParsedArgs) > 1 && data.ParsedArgs[1] != nil {
				durPtr := data.ParsedArgs[1].Duration()
				if durPtr != nil {
					duration = *durPtr
					durationProvided = true
				}
			}

			if durationProvided && duration < 10*time.Minute {
				duration = 10 * time.Minute
			}

			muteReason := "(No reason provided)"
			if len(data.ParsedArgs) > 2 && data.ParsedArgs[2] != nil {
				muteReason = data.ParsedArgs[2].String()
			}

			err = muteUser(config, target.ID, duration)
			if err != nil {
				return errors.New("Something went wrong. Is the bot role above the mute role?")
			}

			err = createCase(config, data.Author, target, logMute, data.ChannelID, muteReason, duration)
			if err != nil {
				unmuteUser(config, data.Author.ID, target.ID)
				return fmt.Errorf("Something went wrong creating the case: %s", err.Error())
			}

			muteEmbed := buildDMEmbed(config, target, logMute, muteReason, duration)
			err = functions.SendDM(target.ID, &discordgo.MessageSend{Embed: muteEmbed})
			if err != nil {
				functions.SendBasicMessage(data.ChannelID, "Was not able to DM the user.")
			}

			ok, delay := triggerDeletion(config)
			if ok {
				functions.DeleteMessage(data.ChannelID, data.Message.ID, time.Duration(delay)*time.Second)
			}

			responseEmbed := responseEmbed(data.Author, target, logWarn)
			message, _ := functions.SendMessage(data.ChannelID, &discordgo.MessageSend{Embed: responseEmbed})
			ok, delay = responseDeletion(config)
			if ok {
				functions.DeleteMessage(data.ChannelID, message.ID, time.Duration(delay)*time.Second)
			}

			return nil
		},
	},
	{
		Command:          "unmute",
		Category:         dcommand.CategoryModeration,
		Aliases:          []string{""},
		Description:      "Unmutes a user for a specified reason",
		RequiredBotPerms: []int64{discordgo.PermissionAdministrator, discordgo.PermissionManageRoles},
		ArgsRequired:     1,
		Args: []*dcommand.Arg{
			{Name: "User", Type: dcommand.User},
			{Name: "Reason", Type: dcommand.String},
		},
		Run: func(data *dcommand.Data) error {
			config, err := moderationBase(data.GuildID)
			if err != nil {
				return err
			}

			target := data.ParsedArgs[0].User()
			err = moderationBasePermissionCheck(data, target, discordgo.PermissionModerateMembers, config.MuteRequiredRoles)
			if err != nil {
				return err
			}

			_, err = functions.GetMember(data.GuildID, target.ID)
			if err != nil {
				return errors.New("Member not found")
			}

			muteRole := config.MuteRole
			_, err = functions.GetRole(config.GuildID, muteRole)
			if muteRole == "" || err != nil {
				return errors.New("No mute role has been setup. Please set one up on the dashboard.")
			}

			err = unmuteUser(config, data.Author.ID, target.ID)
			if err != nil {
				return errors.New("Something went wrong. Is the bot role above the mute role?")
			}

			unmuteReason := "(No reason provided)"
			if len(data.ParsedArgs) > 1 && data.ParsedArgs[1] != nil {
				unmuteReason = data.ParsedArgs[1].String()
			}

			err = createCase(config, data.Author, target, logUnmute, data.ChannelID, unmuteReason)
			if err != nil {
				return fmt.Errorf("Something went wrong creating the case: %s", err.Error())
			}

			unmuteEmbed := buildDMEmbed(config, target, logUnmute, unmuteReason)
			err = functions.SendDM(target.ID, &discordgo.MessageSend{Embed: unmuteEmbed})
			if err != nil {
				functions.SendBasicMessage(data.ChannelID, "Was not able to DM the user.")
			}

			ok, delay := triggerDeletion(config)
			if ok {
				functions.DeleteMessage(data.ChannelID, data.Message.ID, time.Duration(delay)*time.Second)
			}

			responseEmbed := responseEmbed(data.Author, target, logUnmute)
			message, _ := functions.SendMessage(data.ChannelID, &discordgo.MessageSend{Embed: responseEmbed})
			ok, delay = responseDeletion(config)
			if ok {
				functions.DeleteMessage(data.ChannelID, message.ID, time.Duration(delay)*time.Second)
			}

			return nil
		},
	},
	{
		Command:          "kick",
		Category:         dcommand.CategoryModeration,
		Aliases:          []string{""},
		Description:      "Kicks a user for a specified reason",
		RequiredBotPerms: []int64{discordgo.PermissionAdministrator, discordgo.PermissionKickMembers},
		ArgsRequired:     1,
		Args: []*dcommand.Arg{
			{Name: "User", Type: dcommand.User},
			{Name: "Reason", Type: dcommand.String},
		},
		Run: func(data *dcommand.Data) error {
			config, err := moderationBase(data.GuildID)
			if err != nil {
				return err
			}

			target := data.ParsedArgs[0].User()
			err = moderationBasePermissionCheck(data, target, discordgo.PermissionKickMembers, config.MuteRequiredRoles)
			if err != nil {
				return err
			}

			_, err = functions.GetMember(data.GuildID, target.ID)
			if err != nil {
				return errors.New("Member not found")
			}

			kickReason := "(No reason provided)"
			if len(data.ParsedArgs) > 1 && data.ParsedArgs[1] != nil {
				kickReason = data.ParsedArgs[1].String()
			}

			err = createCase(config, data.Author, target, logKick, data.ChannelID, kickReason)
			if err != nil {
				return fmt.Errorf("Something went wrong creating the case: %s", err.Error())
			}

			kickEmbed := buildDMEmbed(config, target, logKick, kickReason)
			err = functions.SendDM(target.ID, &discordgo.MessageSend{Embed: kickEmbed})
			if err != nil {
				functions.SendBasicMessage(data.ChannelID, "Was not able to DM the user.")
			}

			err = kickUser(config, data.Author, target, kickReason)
			if err != nil {
				return errors.New("Something went wrong kicking the user.")
			}

			ok, delay := triggerDeletion(config)
			if ok {
				functions.DeleteMessage(data.ChannelID, data.Message.ID, time.Duration(delay)*time.Second)
			}

			responseEmbed := responseEmbed(data.Author, target, logKick)
			message, _ := functions.SendMessage(data.ChannelID, &discordgo.MessageSend{Embed: responseEmbed})
			ok, delay = responseDeletion(config)
			if ok {
				functions.DeleteMessage(data.ChannelID, message.ID, time.Duration(delay)*time.Second)
			}

			return nil
		},
	},
	{
		Command:          "ban",
		Category:         dcommand.CategoryModeration,
		Aliases:          []string{""},
		Description:      "Bans a user for specified duration and reason",
		RequiredBotPerms: []int64{discordgo.PermissionAdministrator, discordgo.PermissionBanMembers},
		Args: []*dcommand.Arg{
			{Name: "User", Type: dcommand.User},
			{Name: "Duration", Type: dcommand.Duration},
			{Name: "Reason", Type: dcommand.String},
		},
		ArgumentCombos: [][]int{
			{0, 1, 2},
			{0, 2, 1},
			{0, 1},
			{0, 2},
			{0},
		},
		Run: func(data *dcommand.Data) error {
			config, err := moderationBase(data.GuildID)
			if err != nil {
				return err
			}

			target := data.ParsedArgs[0].User()
			err = moderationBasePermissionCheck(data, target, discordgo.PermissionModerateMembers, config.MuteRequiredRoles)
			if err != nil {
				return err
			}

			var duration time.Duration
			var durationProvided bool

			if len(data.ParsedArgs) > 1 && data.ParsedArgs[1] != nil {
				durPtr := data.ParsedArgs[1].Duration()
				if durPtr != nil {
					duration = *durPtr
					durationProvided = true
				}
			}

			if durationProvided && duration < 10*time.Minute {
				duration = 10 * time.Minute
			}

			banReason := "(No reason provided)"
			if len(data.ParsedArgs) > 2 && data.ParsedArgs[2] != nil {
				banReason = data.ParsedArgs[2].String()
			}

			banEmbed := buildDMEmbed(config, target, logBan, banReason, duration)
			err = functions.SendDM(target.ID, &discordgo.MessageSend{Embed: banEmbed})
			if err != nil {
				functions.SendBasicMessage(data.ChannelID, "Was not able to DM the user.")
			}

			err = banUser(config, data.Author, target, banReason, duration)
			if err != nil {
				return fmt.Errorf("Something went wrong: %s", err.Error())
			}

			err = createCase(config, data.Author, target, logBan, data.ChannelID, banReason, duration)
			if err != nil {
				return fmt.Errorf("Something went wrong creating the case: %s", err.Error())
			}

			ok, delay := triggerDeletion(config)
			if ok {
				functions.DeleteMessage(data.ChannelID, data.Message.ID, time.Duration(delay)*time.Second)
			}

			responseEmbed := responseEmbed(data.Author, target, logBan)
			message, _ := functions.SendMessage(data.ChannelID, &discordgo.MessageSend{Embed: responseEmbed})
			ok, delay = responseDeletion(config)
			if ok {
				functions.DeleteMessage(data.ChannelID, message.ID, time.Duration(delay)*time.Second)
			}

			return nil
		},
	},
	{
		Command:          "unban",
		Category:         dcommand.CategoryModeration,
		Aliases:          []string{""},
		Description:      "Unbans a user for a specified reason",
		RequiredBotPerms: []int64{discordgo.PermissionAdministrator, discordgo.PermissionBanMembers},
		ArgsRequired:     1,
		Args: []*dcommand.Arg{
			{Name: "User", Type: dcommand.User},
			{Name: "Reason", Type: dcommand.String},
		},
		Run: func(data *dcommand.Data) error {
			config, err := moderationBase(data.GuildID)
			if err != nil {
				return err
			}

			target := data.ParsedArgs[0].User()
			err = moderationBasePermissionCheck(data, target, discordgo.PermissionBanMembers, config.MuteRequiredRoles)
			if err != nil {
				return err
			}

			unbanReason := data.ParsedArgs[1].String()

			err = unbanUser(config, data.Author.ID, target.ID)
			if err != nil {
				return fmt.Errorf("Something went wrong: %s", err.Error())
			}

			err = createCase(config, data.Author, target, logUnban, data.ChannelID, unbanReason)
			if err != nil {
				return fmt.Errorf("Something went wrong creating the case: %s", err.Error())
			}

			ok, delay := triggerDeletion(config)
			if ok {
				functions.DeleteMessage(data.ChannelID, data.Message.ID, time.Duration(delay)*time.Second)
			}

			responseEmbed := responseEmbed(data.Author, target, logUnban)
			message, _ := functions.SendMessage(data.ChannelID, &discordgo.MessageSend{Embed: responseEmbed})
			ok, delay = responseDeletion(config)
			if ok {
				functions.DeleteMessage(data.ChannelID, message.ID, time.Duration(delay)*time.Second)
			}

			return nil
		},
	},
}

var moderationHelpers = []*dcommand.SummitCommand{
	{
		Command:      "clean",
		Category:     dcommand.CategoryModeration,
		Aliases:      []string{"cl", "purge"},
		Description:  "Delete the last number of messages from the channel with an optional user",
		ArgsRequired: 1,
		Args: []*dcommand.Arg{
			{Name: "Num to delete", Type: &dcommand.IntArg{Min: 1, Max: 100}},
			{Name: "User", Type: dcommand.User},
		},
		Run: func(data *dcommand.Data) error {
			config, err := moderationBase(data.GuildID)
			if err != nil {
				return err
			}

			deleteNum := data.ParsedArgs[0].Int64() + 1

			var user *discordgo.User
			if len(data.ParsedArgs) > 1 {
				user = data.ParsedArgs[1].User()
			}

			messages, err := common.Session.ChannelMessages(data.ChannelID, int(deleteNum), "", "", "")
			if err != nil {
				return errors.New(err.Error())
			}

			var filteredMessages []string
			now := time.Now()
			for _, message := range messages {
				if now.Sub(message.Timestamp) > (14 * time.Hour * 24) {
					continue
				}

				if message.ID == data.Message.ID {
					continue
				}

				if user != nil && message.Author.ID == user.ID {
					filteredMessages = append(filteredMessages, message.ID)
				} else if user == nil {
					filteredMessages = append(filteredMessages, message.ID)
				}
			}

			err = common.Session.ChannelMessagesBulkDelete(data.ChannelID, filteredMessages)
			if err != nil {
				return errors.New(err.Error())
			}

			ok, delay := triggerDeletion(config)
			if ok {
				functions.DeleteMessage(data.ChannelID, data.Message.ID, time.Duration(delay)*time.Second)
			}

			message, _ := functions.SendBasicMessage(data.ChannelID, fmt.Sprintf("Done! Deleted %d messages.", len(filteredMessages)))
			ok, delay = responseDeletion(config)
			if ok {
				functions.DeleteMessage(data.ChannelID, message.ID, time.Duration(delay)*time.Second)
			}

			return nil
		},
	},
}
