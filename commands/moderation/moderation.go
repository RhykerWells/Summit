package moderation

import (
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/dispatch"
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
}
