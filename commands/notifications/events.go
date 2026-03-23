package notifications

import (
	"github.com/RhykerWells/Summit/bot/events"
	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/common/templates"
	"github.com/bwmarrin/discordgo"
)

// initEvents registers all the required event handlers to run on websocket events
func initEvents() {
	events.RegisterGuildMemberJoinfunctions([]func(g *discordgo.GuildMemberAdd){
		guildSendJoinNotification,
	})
	events.RegisterGuildMemberLeavefunctions([]func(g *discordgo.GuildMemberRemove){
		guildSendLeaveNotification,
	})
}

func guildSendJoinNotification(g *discordgo.GuildMemberAdd) {
	config := GetConfig(g.GuildID)

	if config.JoinServerChannel == "" || config.JoinServerMessage == "" {
		return
	}

	guild, err := common.Session.State.Guild(g.GuildID)
	if err != nil {
		return
	}
	channel, err := common.Session.State.Channel(config.JoinServerChannel)
	if err != nil {
		return
	}
	member, err := common.Session.State.Member(guild.ID, g.User.ID)
	if err != nil {
		return
	}

	ctx := templates.NewContext(guild, channel, member)
	renderedNotification, err := ctx.Execute("joinServerMessage", config.JoinServerMessage)
	if err != nil {
		return
	}

	functions.SendBasicMessage(channel.ID, renderedNotification)
}

func guildSendLeaveNotification(g *discordgo.GuildMemberRemove) {
	config := GetConfig(g.GuildID)

	if config.LeaveServerChannel == "" || config.LeaveServerMessage == "" {
		return
	}

	guild, err := common.Session.State.Guild(g.GuildID)
	if err != nil {
		return
	}
	channel, err := common.Session.State.Channel(config.JoinServerChannel)
	if err != nil {
		return
	}
	member, err := common.Session.State.Member(guild.ID, g.User.ID)
	if err != nil {
		return
	}

	ctx := templates.NewContext(guild, channel, member)
	renderedNotification, err := ctx.Execute("leaveServerMessage", config.LeaveServerMessage)
	if err != nil {
		return
	}

	functions.SendBasicMessage(channel.ID, renderedNotification)
}
