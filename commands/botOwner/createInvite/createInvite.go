package createinvite

import (
	"errors"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/commands/util"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/common/dcommand"
	"github.com/bwmarrin/discordgo"
)

var Command = &dcommand.SummitCommand{
	Command:      "createinvite",
	Category:     dcommand.CategoryOwner,
	Description:  "Creates an invite to the specified guild",
	ArgsRequired: 1,
	Args: []*dcommand.Arg{
		{Name: "GuildID", Type: dcommand.String},
	},
	Run: util.OwnerCommand(func(data *dcommand.Data) error {
		channels, _ := common.Session.GuildChannels(data.ParsedArgs[0].String())
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
		functions.SendBasicMessage(data.ChannelID, "discord.gg/"+invite.Code)

		return nil
	}),
}
