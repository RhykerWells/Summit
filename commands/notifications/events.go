package notifications

import (
	eventsv2 "github.com/RhykerWells/Summit/bot/eventsV2"
	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/common/templates"
	"github.com/bwmarrin/discordgo"
)

// initEvents registers all the required event handlers to run on websocket events
func initEvents() {
	eventsv2.AddHandler(handleJoinLeaveServerMessage, eventsv2.EventGuildMemberAdd, eventsv2.EventGuildMemberRemove)
}

func handleJoinLeaveServerMessage(data *eventsv2.EventData) error {
	var m *discordgo.Member

	switch data.Type {
	case eventsv2.EventGuildMemberAdd:
		m = data.GuildMemberAdd().Member
	case eventsv2.EventGuildMemberRemove:
		m = data.GuildMemberRemove().Member
	default:
		return nil
	}

	config := GetConfig(m.GuildID)

	switch data.Type {
	case eventsv2.EventGuildMemberAdd:
		if config.JoinServerChannel == "" && config.JoinServerMessage == "" {
			return nil
		}
	case eventsv2.EventGuildMemberRemove:
		if config.LeaveServerChannel == "" && config.LeaveServerMessage == "" {
			return nil
		}
	}

	guild, err := common.Session.State.Guild(m.GuildID)
	if err != nil {
		return err
	}
	channel, err := common.Session.State.Channel(config.JoinServerChannel)
	if err != nil {
		return err
	}

	ctx := templates.NewContext(guild, channel, m)
	var renderedNotification string

	switch data.Type {
	case eventsv2.EventGuildMemberAdd:
		renderedNotification, _ = ctx.Execute("joinServerMessage", config.JoinServerMessage)
	case eventsv2.EventGuildMemberRemove:
		renderedNotification, _ = ctx.Execute("leaveServerMessage", config.LeaveServerMessage)
	}

	functions.SendBasicMessage(channel.ID, renderedNotification)

	return nil
}
