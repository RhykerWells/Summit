package createinvite

import (
	"errors"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/dispatch"
	"github.com/bwmarrin/discordgo"
)

var Command = &dispatch.Command{
	Command:      "createinvite",
	Category:     command.CategoryOwner,
	Description:  "Creates an invite to the specified guild",
	ArgsRequired: 1,
	Args: []*dispatch.Arg{
		{Name: "GuildID", Type: dispatch.String},
	},
	Run: command.OwnerCommand(func(data *dispatch.Data) error {
		guildID := data.ParsedArgs[0].Value.(string)
		channels, _ := common.Session.GuildChannels(guildID)
		var channelID string
		for _, v := range channels {
			if v.Type == discordgo.ChannelTypeGuildText {
				channelID = v.ID
				break
			}
		}
		if channelID == "0" {
			return errors.New("No available channels")
		}

		invite, _ := common.Session.ChannelInviteCreate(channelID, discordgo.Invite{
			MaxAge:    120,
			MaxUses:   1,
			Temporary: true,
			Unique:    true,
		})
		functions.SendBasicMessage(data.Channel.ID, "discord.gg/"+invite.Code)

		return nil
	}),
}
