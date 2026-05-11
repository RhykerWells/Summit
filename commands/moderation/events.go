package moderation

import (
	"context"
	"errors"

	"github.com/RhykerWells/Summit/bot/events"
	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/commands/moderation/models"
	"github.com/RhykerWells/Summit/common"
	"github.com/bwmarrin/discordgo"
)

// initEvents registers all the required event handlers to run on websocket events
func initEvents() {
	events.RegisterGuildJoinfunctions([]func(g *discordgo.GuildCreate){
		guildAddModerationConfig,
	})

	events.RegisterGuildLeavefunctions([]func(g *discordgo.GuildDelete){
		guildDeleteModerationConfig,
	})

	events.RegisterChannelCreatefunctions([]func(c *discordgo.ChannelCreate){
		func(c *discordgo.ChannelCreate) {
			refreshMuteSettingsOnChannel(GetConfig(c.GuildID), c.Channel)
		},
	})

	events.RegisterChannelUpdatefunctions([]func(c *discordgo.ChannelUpdate){
		func(c *discordgo.ChannelUpdate) {
			refreshMuteSettingsOnChannel(GetConfig(c.GuildID), c.Channel)
		},
	})

	events.RegisterAuditLogCreateFunctions([]func(g *discordgo.GuildAuditLogEntryCreate){
		logGuildModerationNotByBot,
	})
}

// guildAddModerationConfig creates the intial configs for the moderation system for a specified guild
func guildAddModerationConfig(g *discordgo.GuildCreate) {
	config := GetConfig(g.ID)
	SaveConfig(config)
}

// guildDeleteModerationConfig deletes the config for the moderation system for a specified guild
func guildDeleteModerationConfig(g *discordgo.GuildDelete) {
	config, err := models.ModerationConfigs(models.ModerationConfigWhere.GuildID.EQ(g.ID)).One(context.Background(), common.PQ)
	if err != nil {
		return
	}

	config.Delete(context.Background(), common.PQ)
}

func logGuildModerationNotByBot(g *discordgo.GuildAuditLogEntryCreate) {
	entry := g.AuditLogEntry

	config := GetConfig(g.GuildID)

	err := auditLogCheckBase(entry, config)
	if err != nil {
		return
	}

	author, _ := functions.GetUser(g.UserID)
	target, _ := functions.GetUser(entry.TargetID)

	switch *entry.ActionType {
	case discordgo.AuditLogActionMemberBanAdd:
		createCase(config, author, target, logBan, nil, entry.Reason, nil)
	case discordgo.AuditLogActionMemberBanRemove:
		createCase(config, author, target, logUnban, nil, entry.Reason, nil)
	case discordgo.AuditLogActionMemberKick:
		createCase(config, author, target, logKick, nil, entry.Reason, nil)
	}
}

func auditLogCheckBase(entry *discordgo.AuditLogEntry, config *Config) error {
	if !config.ModerationEnabled {
		return errors.New("the moderation system is not enabled")
	}

	if config.ModerationLogChannel == "" {
		return errors.New("no log channel")
	}

	switch *entry.ActionType {
	case discordgo.AuditLogActionMemberBanAdd, discordgo.AuditLogActionMemberBanRemove, discordgo.AuditLogActionMemberKick:
	default:
		return errors.New("not a moderation action")
	}

	if entry.UserID == common.Bot.ID {
		return errors.New("handled via moderation system")
	}

	if entry.UserID == "" || entry.TargetID == "" {
		return errors.New("invalid user or target")
	}

	return nil
}
