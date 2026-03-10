package invite

import (
	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/dispatch"
)

var Command = &dispatch.Command{
	Command:     "invite",
	Category:    command.CategoryGeneral,
	Description: "Creates an invite link for the bot",
	Run: (func(data *dispatch.Data) error {
		functions.SendBasicMessage(data.Channel.ID, "[Invite link](<https://discord.com/oauth2/authorize?client_id="+common.ConfigBotClientID+">)")
		return nil
	}),
}
