package moderation

import (
	"github.com/RhykerWells/Summit/bot/events"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/dispatch"
	"github.com/bwmarrin/discordgo"
)

//go:generate sqlboiler --no-hooks psql

var (
	MessageStore *MessageCache
)

// ModerationSetup runs the following:
//   - The schema initialiser
//   - Initialises event handlers
//   - Initialises the web plugin
//   - Initialises any other required middlewares
//   - Registration of the moderation commands & their pagination
func ModerationSetup(cmdHandler *dispatch.CommandHandler) {
	common.InitSchema("Moderation", GuildModerationSchema...)

	initEvents()

	initWeb()

	scheduleAllPendingUnmutes()
	scheduleAllPendingUnbans()

	// Moderation commands
	cmdHandler.RegisterCommands(moderationCommands...)

	// Moderation helpers
	cmdHandler.RegisterCommands(moderationHelpers...)

	MessageStore = NewMessageCache(100)

	events.RegisterMessageCreateFunctions([]func(*discordgo.MessageCreate){
		handleMessageCreate,
	})

	events.RegisterMessageDeleteFunctions([]func(*discordgo.MessageDelete){
		handleMessageDelete,
	})
}

func handleMessageCreate(m *discordgo.MessageCreate) {
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
}

func handleMessageDelete(m *discordgo.MessageDelete) {
	MessageStore.Delete(m.ChannelID, m.ID)
}
