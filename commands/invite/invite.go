package invite

import (
	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/common/dcommand"
)

var Command = &dcommand.SummitCommand{
	Command:     "invite",
	Category:    dcommand.CategoryGeneral,
	Description: "Creates an invite link for the bot",
	Run: (func(data *dcommand.Data) error {
		functions.SendBasicMessage(data.ChannelID, "[Invite link](<https://discord.com/oauth2/authorize?client_id="+common.ConfigBotClientID+">)")
		return nil
	}),
}
