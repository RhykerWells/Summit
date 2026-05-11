package events

import (
	"database/sql"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

var db *sql.DB

// InitEvents registers the required event handlers to pass data to the bot
func InitEvents(s *discordgo.Session, database *sql.DB) {
	db = database

	s.AddHandler(botReady)

	// Guild events
	s.AddHandler(guildJoin)
	s.AddHandler(guildLeave)
	s.AddHandler(auditLogCreate)

	// Guild role events
	s.AddHandler(guildRoleCreate)
	s.AddHandler(guildRoleUpdate)
	s.AddHandler(guildRoleDelete)

	// Guild channel events
	s.AddHandler(channelCreate)
	s.AddHandler(channelUpdate)
	s.AddHandler(channelDelete)

	// Message events
	s.AddHandler(messageCreate)
	s.AddHandler(messageUpdate)
	s.AddHandler(messageDelete)

	// Guild member events
	s.AddHandler(guildMemberAdd)
	s.AddHandler(guildMemberLeave)

	log.Infoln("Event system initialised")
}
