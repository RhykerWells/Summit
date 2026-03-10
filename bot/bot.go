package bot

import (
	"database/sql"

	"github.com/RhykerWells/Summit/bot/core"
	"github.com/RhykerWells/Summit/bot/events"
	"github.com/RhykerWells/Summit/command"
	"github.com/bwmarrin/discordgo"
)

var (
	gatewayIntentsUsed = discordgo.MakeIntent(
		discordgo.IntentGuilds |
			discordgo.IntentGuildMembers |
			discordgo.IntentGuildModeration |
			discordgo.IntentGuildVoiceStates |
			discordgo.IntentGuildPresences |
			discordgo.IntentGuildMessages |
			discordgo.IntentGuildMessageReactions |
			discordgo.IntentDirectMessages |
			discordgo.IntentDirectMessageReactions |
			discordgo.IntentMessageContent |
			discordgo.IntentGuildScheduledEvents,
	)
)

// Run initialises all the core bot modules such as the event system
// the core bot config, the command system and the intents the bot needs
func Run(s *discordgo.Session, db *sql.DB) {
	events.InitEvents(s, db)
	core.Init()
	command.InitCommandHandler(s)
	s.Identify.Intents = gatewayIntentsUsed
}
