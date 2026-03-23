package notifications

import (
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/dispatch"
)

//go:generate sqlboiler --no-hooks psql

// NotificationSetup runs the following:
//   - The schema initialiser
//   - Registration of the guild and user join/leave functions
//   - Initialises the web plugin
func NotificationSetup(cmdHandler *dispatch.CommandHandler) {
	common.InitSchema("Notifications", GuildNotificationSchema...)
	initEvents()
	initWeb()
}
