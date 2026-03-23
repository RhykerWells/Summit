package commands

import (
	"github.com/bwmarrin/discordgo"

	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/Summit/commands/economy"
	banserver "github.com/RhykerWells/Summit/commands/maintenance/banServer"
	createinvite "github.com/RhykerWells/Summit/commands/maintenance/createInvite"
	leaveserver "github.com/RhykerWells/Summit/commands/maintenance/leaveServer"
	setstatus "github.com/RhykerWells/Summit/commands/maintenance/setStatus"
	unbanserver "github.com/RhykerWells/Summit/commands/maintenance/unbanServer"
	"github.com/RhykerWells/Summit/commands/moderation"
	"github.com/RhykerWells/Summit/commands/notifications"
	"github.com/RhykerWells/Summit/commands/standard/help"
	"github.com/RhykerWells/Summit/commands/standard/invite"
	"github.com/RhykerWells/Summit/commands/standard/ping"
	"github.com/RhykerWells/Summit/commands/standard/prefix"
)

// InitCommands registers all available commands, and attaches the handler to the Discord session.
//
// After registration, the handler is connected to the session so that
// incoming message events are processed and routed to the correct
// command.
func InitCommands(session *discordgo.Session) {
	command.CommandHandler.RegisterCommands(
		help.Command,
		prefix.Command,

		ping.Command,
		invite.Command,

		banserver.Command,
		createinvite.Command,
		leaveserver.Command,
		setstatus.Command,
		unbanserver.Command,
	)

	economy.EconomySetup(command.CommandHandler)
	moderation.ModerationSetup(command.CommandHandler)
	notifications.NotificationSetup(command.CommandHandler)
	session.AddHandler(command.CommandHandler.HandleMessageCreate)
}
