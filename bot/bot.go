package bot

import (
	"context"

	"github.com/RhykerWells/Summit/bot/core/models"
	eventsv2 "github.com/RhykerWells/Summit/bot/eventsV2"
	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/Summit/commands"
	"github.com/RhykerWells/Summit/common"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
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

func Run() {
	logrus.Infoln("Starting bot")
	common.Session.Identify.Intents = gatewayIntentsUsed

	common.Session.AddHandler(eventsv2.HandleEvent)
	addAdditionalHandlers()

	err := common.Session.Open()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to open Discord session")
	}
	common.Bot = common.Session.State.User

	logrus.Infoln("Initialising command handler and commands")
	command.InitCommandHandler(common.Session)
	commands.InitCommands(common.Session)

	common.InitPluginCommands()

	common.InitBotPlugins()
}

func addAdditionalHandlers() {
	eventsv2.AddHandler(handleBotReady, eventsv2.EventReady)

	eventsv2.AddHandler(handleGuildJoin, eventsv2.EventGuildCreate)
	eventsv2.AddHandler(handleGuildDelete, eventsv2.EventGuildDelete)
}

func handleBotReady(data *eventsv2.EventData) error {
	r := data.Ready()
	guildCount := len(r.Guilds)

	logrus.Infof("Connected to: %d guilds", guildCount)
	functions.SetStatus(common.VERSION)

	return nil
}

func handleGuildJoin(data *eventsv2.EventData) error {
	g := data.GuildCreate()

	_, err := models.FindBannedGuildG(context.Background(), g.ID)
	if err == nil {
		logrus.WithFields(logrus.Fields{
			"guild": g.ID,
			"owner": g.OwnerID,
		}).Warnln("Banned guild attempted to add bot: ", g.Name)

		data.Session.GuildLeave(g.ID)
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"guild":       g.ID,
		"owner":       g.OwnerID,
		"membercount": g.MemberCount,
	}).Infoln("Joined guild: ", g.Name)

	return nil
}

func handleGuildDelete(data *eventsv2.EventData) error {
	g := data.GuildDelete()

	if g.Unavailable {
		return nil
	}

	logrus.Infoln("Left guild: ", g.ID)

	return nil
}
