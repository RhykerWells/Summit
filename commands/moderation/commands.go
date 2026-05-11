package moderation

import (
	"errors"
	"fmt"
	"time"

	"slices"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/dispatch"
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

func moderationBasePermissionCheck(data *dispatch.Data, target *discordgo.User, discordPermissionRequired int64, additionalPermittanceRoles []string) error {
	perms, _ := common.Session.State.UserChannelPermissions(data.Author.ID, data.Channel.ID)

	permsMet := false
	if perms&discordPermissionRequired != 0 {
		permsMet = true
	}

	if !permsMet && additionalPermittanceRoles != nil {
		permsMet = hasRequiredRole(data.Guild.ID, data.Author.ID, additionalPermittanceRoles)
	}

	if !permsMet {
		humanisedPerm, _ := dispatch.PermissionNames[discordPermissionRequired]
		return fmt.Errorf("This command requires either the `%s` permission or an additional role set up by admins through the dashboard.", humanisedPerm)
	}

	targetMember, _ := functions.GetMember(data.Guild.ID, target.ID)
	if targetMember != nil {
		author, _ := functions.GetMember(data.Guild.ID, data.Author.ID)
		ok := functions.IsMemberHigher(data.Guild.ID, author, targetMember)
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

var moderationCommands = []*dispatch.Command{
	{
		Command:      "warn",
		Category:     command.CategoryModeration,
		Aliases:      []string{""},
		Description:  "Warns a user for a specified reason",
		ArgsRequired: 2,
		Args: []*dispatch.Arg{
			{Name: "User", Type: dispatch.User},
			{Name: "Reason", Type: dispatch.String},
		},
		Run: func(data *dispatch.Data) error {
			config, err := moderationBase(data.Guild.ID)
			if err != nil {
				return err
			}

			target := data.ParsedArgs[0].Value.(*discordgo.User)
			err = moderationBasePermissionCheck(data, target, discordgo.PermissionModerateMembers, config.WarnRequiredRoles)
			if err != nil {
				return err
			}

			_, err = functions.GetMember(data.Guild.ID, target.ID)
			if err != nil {
				return errors.New("Member not found")
			}

			warnReason := data.ParsedArgs[1].Value.(string)

			err = createCase(config, data.Author, target, logWarn, data.Channel, warnReason, nil)
			if err != nil {
				return fmt.Errorf("Something went wrong creating the case: %s", err.Error())
			}

			warnEmbed := buildDMEmbed(config, target, logWarn, warnReason, nil)
			err = functions.SendDM(target.ID, &discordgo.MessageSend{Embed: warnEmbed})
			if err != nil {
				functions.SendBasicMessage(data.Channel.ID, "Was not able to DM the user.")
			}

			ok, delay := triggerDeletion(config)
			if ok {
				functions.DeleteMessage(data.Channel.ID, data.Message.ID, time.Duration(delay)*time.Second)
			}

			responseEmbed := responseEmbed(data.Author, target, logWarn)
			message, _ := functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: responseEmbed})
			ok, delay = responseDeletion(config)
			if ok {
				functions.DeleteMessage(data.Channel.ID, message.ID, time.Duration(delay)*time.Second)
			}

			return nil
		},
	},
	{
		Command:          "mute",
		Category:         command.CategoryModeration,
		Aliases:          []string{""},
		Description:      "Mutes a user for specified duration and reason",
		RequiredBotPerms: []int64{discordgo.PermissionAdministrator, discordgo.PermissionManageRoles},
		ArgsRequired:     3,
		Args: []*dispatch.Arg{
			{Name: "User", Type: dispatch.User},
			{Name: "Duration", Type: dispatch.Duration},
			{Name: "Reason", Type: dispatch.String},
		},
		ArgumentCombos: [][]int{
			{0, 1, 2},
			{0, 2, 1},
			{0, 1},
			{0, 2},
			{0},
		},
		Run: func(data *dispatch.Data) error {
			config, err := moderationBase(data.Guild.ID)
			if err != nil {
				return err
			}

			target := data.ParsedArgs[0].Value.(*discordgo.User)
			err = moderationBasePermissionCheck(data, target, discordgo.PermissionModerateMembers, config.MuteRequiredRoles)
			if err != nil {
				return err
			}

			_, err = functions.GetMember(data.Guild.ID, target.ID)
			if err != nil {
				return errors.New("Member not found")
			}

			var durationPtr *time.Duration

			if len(data.ParsedArgs) > 1 && data.ParsedArgs[1] != nil {
				duration := data.ParsedArgs[1].Value.(time.Duration)
				if duration < 10*time.Minute {
					duration = 10 * time.Minute
				}

				durationPtr = &duration
			}

			muteReason := "(No reason provided)"
			if len(data.ParsedArgs) > 2 && data.ParsedArgs[2] != nil {
				muteReason = data.ParsedArgs[2].Value.(string)
			}

			err = muteUser(config, target.ID, durationPtr)
			if err != nil {
				return errors.New("Something went wrong. Is the bot role above the mute role?")
			}

			err = createCase(config, data.Author, target, logMute, data.Channel, muteReason, durationPtr)
			if err != nil {
				unmuteUser(config, data.Author.ID, target.ID)
				return fmt.Errorf("Something went wrong creating the case: %s", err.Error())
			}

			muteEmbed := buildDMEmbed(config, target, logMute, muteReason, durationPtr)
			err = functions.SendDM(target.ID, &discordgo.MessageSend{Embed: muteEmbed})
			if err != nil {
				functions.SendBasicMessage(data.Channel.ID, "Was not able to DM the user.")
			}

			ok, delay := triggerDeletion(config)
			if ok {
				functions.DeleteMessage(data.Channel.ID, data.Message.ID, time.Duration(delay)*time.Second)
			}

			responseEmbed := responseEmbed(data.Author, target, logMute)
			message, _ := functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: responseEmbed})
			ok, delay = responseDeletion(config)
			if ok {
				functions.DeleteMessage(data.Channel.ID, message.ID, time.Duration(delay)*time.Second)
			}

			return nil
		},
	},
	{
		Command:          "unmute",
		Category:         command.CategoryModeration,
		Aliases:          []string{""},
		Description:      "Unmutes a user for a specified reason",
		RequiredBotPerms: []int64{discordgo.PermissionAdministrator, discordgo.PermissionManageRoles},
		ArgsRequired:     1,
		Args: []*dispatch.Arg{
			{Name: "User", Type: dispatch.User},
			{Name: "Reason", Type: dispatch.String},
		},
		Run: func(data *dispatch.Data) error {
			config, err := moderationBase(data.Guild.ID)
			if err != nil {
				return err
			}

			target := data.ParsedArgs[0].Value.(*discordgo.User)
			err = moderationBasePermissionCheck(data, target, discordgo.PermissionModerateMembers, config.MuteRequiredRoles)
			if err != nil {
				return err
			}

			_, err = functions.GetMember(data.Guild.ID, target.ID)
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
				unmuteReason = data.ParsedArgs[1].Value.(string)
			}

			err = createCase(config, data.Author, target, logUnmute, data.Channel, unmuteReason, nil)
			if err != nil {
				return fmt.Errorf("Something went wrong creating the case: %s", err.Error())
			}

			unmuteEmbed := buildDMEmbed(config, target, logUnmute, unmuteReason, nil)
			err = functions.SendDM(target.ID, &discordgo.MessageSend{Embed: unmuteEmbed})
			if err != nil {
				functions.SendBasicMessage(data.Channel.ID, "Was not able to DM the user.")
			}

			ok, delay := triggerDeletion(config)
			if ok {
				functions.DeleteMessage(data.Channel.ID, data.Message.ID, time.Duration(delay)*time.Second)
			}

			responseEmbed := responseEmbed(data.Author, target, logUnmute)
			message, _ := functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: responseEmbed})
			ok, delay = responseDeletion(config)
			if ok {
				functions.DeleteMessage(data.Channel.ID, message.ID, time.Duration(delay)*time.Second)
			}

			return nil
		},
	},
	{
		Command:          "kick",
		Category:         command.CategoryModeration,
		Aliases:          []string{""},
		Description:      "Kicks a user for a specified reason",
		RequiredBotPerms: []int64{discordgo.PermissionAdministrator, discordgo.PermissionKickMembers},
		ArgsRequired:     1,
		Args: []*dispatch.Arg{
			{Name: "User", Type: dispatch.User},
			{Name: "Reason", Type: dispatch.String},
		},
		Run: func(data *dispatch.Data) error {
			config, err := moderationBase(data.Guild.ID)
			if err != nil {
				return err
			}

			target := data.ParsedArgs[0].Value.(*discordgo.User)
			err = moderationBasePermissionCheck(data, target, discordgo.PermissionKickMembers, config.KickRequiredRoles)
			if err != nil {
				return err
			}

			_, err = functions.GetMember(data.Guild.ID, target.ID)
			if err != nil {
				return errors.New("Member not found")
			}

			kickReason := "(No reason provided)"
			if len(data.ParsedArgs) > 1 && data.ParsedArgs[1] != nil {
				kickReason = data.ParsedArgs[1].Value.(string)
			}

			err = createCase(config, data.Author, target, logKick, data.Channel, kickReason, nil)
			if err != nil {
				return fmt.Errorf("Something went wrong creating the case: %s", err.Error())
			}

			kickEmbed := buildDMEmbed(config, target, logKick, kickReason, nil)
			err = functions.SendDM(target.ID, &discordgo.MessageSend{Embed: kickEmbed})
			if err != nil {
				functions.SendBasicMessage(data.Channel.ID, "Was not able to DM the user.")
			}

			err = kickUser(config, data.Author, target, kickReason)
			if err != nil {
				return errors.New("Something went wrong kicking the user.")
			}

			ok, delay := triggerDeletion(config)
			if ok {
				functions.DeleteMessage(data.Channel.ID, data.Message.ID, time.Duration(delay)*time.Second)
			}

			responseEmbed := responseEmbed(data.Author, target, logKick)
			message, _ := functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: responseEmbed})
			ok, delay = responseDeletion(config)
			if ok {
				functions.DeleteMessage(data.Channel.ID, message.ID, time.Duration(delay)*time.Second)
			}

			return nil
		},
	},
	{
		Command:          "ban",
		Category:         command.CategoryModeration,
		Aliases:          []string{""},
		Description:      "Bans a user for specified duration and reason",
		RequiredBotPerms: []int64{discordgo.PermissionAdministrator, discordgo.PermissionBanMembers},
		Args: []*dispatch.Arg{
			{Name: "User", Type: dispatch.User},
			{Name: "Duration", Type: dispatch.Duration},
			{Name: "Reason", Type: dispatch.String},
		},
		ArgumentCombos: [][]int{
			{0, 1, 2},
			{0, 2, 1},
			{0, 1},
			{0, 2},
			{0},
		},
		Run: func(data *dispatch.Data) error {
			config, err := moderationBase(data.Guild.ID)
			if err != nil {
				return err
			}

			target := data.ParsedArgs[0].Value.(*discordgo.User)
			err = moderationBasePermissionCheck(data, target, discordgo.PermissionBanMembers, config.BanRequiredRoles)
			if err != nil {
				return err
			}

			var durationPtr *time.Duration

			if len(data.ParsedArgs) > 1 && data.ParsedArgs[1] != nil {
				duration := data.ParsedArgs[1].Value.(time.Duration)
				if duration < 10*time.Minute {
					duration = 10 * time.Minute
				}

				durationPtr = &duration
			}

			banReason := "(No reason provided)"
			if len(data.ParsedArgs) > 2 && data.ParsedArgs[2] != nil {
				banReason = data.ParsedArgs[2].Value.(string)
			}

			banEmbed := buildDMEmbed(config, target, logBan, banReason, durationPtr)
			err = functions.SendDM(target.ID, &discordgo.MessageSend{Embed: banEmbed})
			if err != nil {
				functions.SendBasicMessage(data.Channel.ID, "Was not able to DM the user.")
			}

			err = banUser(config, data.Author, target, banReason, durationPtr)
			if err != nil {
				return fmt.Errorf("Something went wrong: %s", err.Error())
			}

			err = createCase(config, data.Author, target, logBan, data.Channel, banReason, durationPtr)
			if err != nil {
				return fmt.Errorf("Something went wrong creating the case: %s", err.Error())
			}

			ok, delay := triggerDeletion(config)
			if ok {
				functions.DeleteMessage(data.Channel.ID, data.Message.ID, time.Duration(delay)*time.Second)
			}

			responseEmbed := responseEmbed(data.Author, target, logBan)
			message, _ := functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: responseEmbed})
			ok, delay = responseDeletion(config)
			if ok {
				functions.DeleteMessage(data.Channel.ID, message.ID, time.Duration(delay)*time.Second)
			}

			return nil
		},
	},
	{
		Command:          "unban",
		Category:         command.CategoryModeration,
		Aliases:          []string{""},
		Description:      "Unbans a user for a specified reason",
		RequiredBotPerms: []int64{discordgo.PermissionAdministrator, discordgo.PermissionBanMembers},
		ArgsRequired:     1,
		Args: []*dispatch.Arg{
			{Name: "User", Type: dispatch.User},
			{Name: "Reason", Type: dispatch.String},
		},
		Run: func(data *dispatch.Data) error {
			config, err := moderationBase(data.Guild.ID)
			if err != nil {
				return err
			}

			target := data.ParsedArgs[0].Value.(*discordgo.User)
			err = moderationBasePermissionCheck(data, target, discordgo.PermissionBanMembers, config.BanRequiredRoles)
			if err != nil {
				return err
			}

			unbanReason := data.ParsedArgs[1].Value.(string)

			err = unbanUser(config, data.Author.ID, target.ID)
			if err != nil {
				return fmt.Errorf("Something went wrong: %s", err.Error())
			}

			err = createCase(config, data.Author, target, logUnban, data.Channel, unbanReason, nil)
			if err != nil {
				return fmt.Errorf("Something went wrong creating the case: %s", err.Error())
			}

			ok, delay := triggerDeletion(config)
			if ok {
				functions.DeleteMessage(data.Channel.ID, data.Message.ID, time.Duration(delay)*time.Second)
			}

			responseEmbed := responseEmbed(data.Author, target, logUnban)
			message, _ := functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: responseEmbed})
			ok, delay = responseDeletion(config)
			if ok {
				functions.DeleteMessage(data.Channel.ID, message.ID, time.Duration(delay)*time.Second)
			}

			return nil
		},
	},
}

var moderationHelpers = []*dispatch.Command{
	{
		Command:           "clean",
		Category:          command.CategoryModeration,
		Aliases:           []string{"cl", "purge"},
		Description:       "Delete the last number of messages from the channel with an optional user",
		ArgsRequired:      1,
		RequiredUserPerms: []int64{discordgo.PermissionManageMessages},
		RequiredBotPerms:  []int64{discordgo.PermissionManageMessages},
		Args: []*dispatch.Arg{
			{Name: "Num to delete", Type: &dispatch.IntArg{Min: 1, Max: 100}},
			{Name: "User", Type: dispatch.User},
		},
		Run: func(data *dispatch.Data) error {
			config, err := moderationBase(data.Guild.ID)
			if err != nil {
				return err
			}

			deleteNum := data.ParsedArgs[0].Value.(int) + 1

			var user *discordgo.User
			if len(data.ParsedArgs) > 1 {
				user = data.ParsedArgs[1].Value.(*discordgo.User)
			}

			messages, err := common.Session.ChannelMessages(data.Channel.ID, int(deleteNum), "", "", "")
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

			err = common.Session.ChannelMessagesBulkDelete(data.Channel.ID, filteredMessages)
			if err != nil {
				return errors.New(err.Error())
			}

			ok, delay := triggerDeletion(config)
			if ok {
				functions.DeleteMessage(data.Channel.ID, data.Message.ID, time.Duration(delay)*time.Second)
			}

			message, _ := functions.SendBasicMessage(data.Channel.ID, fmt.Sprintf("Done! Deleted %d messages.", len(filteredMessages)))
			ok, delay = responseDeletion(config)
			if ok {
				functions.DeleteMessage(data.Channel.ID, message.ID, time.Duration(delay)*time.Second)
			}

			return nil
		},
	},
}
