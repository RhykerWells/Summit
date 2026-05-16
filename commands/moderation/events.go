package moderation

import (
	"context"
	"errors"

	eventsv2 "github.com/RhykerWells/Summit/bot/eventsV2"
	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/commands/moderation/models"
	"github.com/RhykerWells/Summit/common"
	"github.com/bwmarrin/discordgo"
)

// initEvents registers all the required event handlers to run on websocket events
func initEvents() {
	eventsv2.AddHandler(handleGuildCreate, eventsv2.EventGuildCreate)
	eventsv2.AddHandler(handleGuildDelete, eventsv2.EventGuildDelete)

	eventsv2.AddHandler(handleAuditLogEntry, eventsv2.EventGuildAuditLogEntryCreate)

	eventsv2.AddHandler(handleChannelCreateUpdate, eventsv2.EventChannelCreate, eventsv2.EventChannelUpdate)

	eventsv2.AddHandler(handleMessageCreate, eventsv2.EventMessageCreate)
	eventsv2.AddHandler(handleMessageDelete, eventsv2.EventMessageDelete, eventsv2.EventMessageDeleteBulk)
}

// handleGuildCreate creates the intial configs for the moderation system for a specified guild
func handleGuildCreate(data *eventsv2.EventData) error {
	g := data.GuildCreate()

	config := GetConfig(g.ID)
	SaveConfig(config)

	return nil
}

// handleGuildDelete deletes the config for the moderation system for a specified guild
func handleGuildDelete(data *eventsv2.EventData) error {
	g := data.GuildDelete()

	config, err := models.ModerationConfigs(models.ModerationConfigWhere.GuildID.EQ(g.ID)).One(context.Background(), common.PQ)
	if err != nil {
		return err
	}

	_, err = config.Delete(context.Background(), common.PQ)
	if err != nil {
		return err
	}

	return nil
}

func handleAuditLogEntry(data *eventsv2.EventData) error {
	entry := data.GuildAuditLogEntryCreate()

	config := GetConfig(entry.GuildID)

	err := auditLogCheckBase(entry.AuditLogEntry, config)
	if err != nil {
		return err
	}

	author, _ := functions.GetUser(entry.UserID)
	target, _ := functions.GetUser(entry.TargetID)

	switch *entry.ActionType {
	case discordgo.AuditLogActionMemberBanAdd:
		createCase(config, author, target, logBan, nil, entry.Reason, nil)
	case discordgo.AuditLogActionMemberBanRemove:
		createCase(config, author, target, logUnban, nil, entry.Reason, nil)
	case discordgo.AuditLogActionMemberKick:
		createCase(config, author, target, logKick, nil, entry.Reason, nil)
	}

	return nil
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

// handleChannelCreateUpdate handles refreshing the mute configuration when channels are updated, to ensure that mute role permissions are active
func handleChannelCreateUpdate(data *eventsv2.EventData) error {
	var c *discordgo.Channel
	switch data.Type {
	case eventsv2.EventChannelCreate:
		c = data.ChannelCreate().Channel
	case eventsv2.EventChannelUpdate:
		c = data.ChannelUpdate().Channel
	}

	config := GetConfig(c.GuildID)

	refreshMuteSettingsOnChannel(config, c)

	return nil
}

func handleMessageCreate(data *eventsv2.EventData) error {
	m := data.MessageCreate()

	guild, err := common.Session.State.Guild(m.GuildID)
	if err != nil {
		guild = &discordgo.Guild{ID: m.GuildID, Name: "Unknown Guild"}
	}

	channel, err := common.Session.State.Channel(m.ChannelID)
	if err != nil {
		channel = &discordgo.Channel{ID: m.ChannelID, Name: "Unknown Channel"}
	}

	MessageStore.Add(&CachedMessage{
		ID:          m.ID,
		Guild:       guild,
		Channel:     channel,
		Author:      m.Author,
		Content:     m.Content,
		Attachments: m.Attachments,
		CreatedAt:   m.Timestamp,
	})

	return nil
}

func handleMessageDelete(data *eventsv2.EventData) error {
	if data.Type == eventsv2.EventMessageDelete {
		m := data.MessageDelete()
		MessageStore.Delete(m.ChannelID, m.ID)

		return nil
	}

	for _, m := range data.MessageDeleteBulk().Messages {
		MessageStore.Delete(data.MessageDeleteBulk().ChannelID, m)
	}

	return nil
}
