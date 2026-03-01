package events

import (
	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/common"
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// botReady is sent when the bot originally connects to the gateway
// This is used to test the bot actually connects
func botReady(s *discordgo.Session, r *discordgo.Ready) {
	guildCount := len(r.Guilds)
	log.Infof("Connected to: %d guilds", guildCount)

	functions.SetStatus(common.VERSION)
}
