package dcommand

import (
	"strings"

	"github.com/RhykerWells/Summit/bot/functions"
	prfx "github.com/RhykerWells/Summit/bot/prefix"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

var CmdHndlr *CommandHandler

// NewCommandHandler creates a new command handler
func NewCommandHandler() *CommandHandler {
	handler := &CommandHandler{
		cmdInstances: make([]SummitCommand, 0),
		cmdMap:       make(map[string]SummitCommand),
	}
	CmdHndlr = handler

	return CmdHndlr
}

// Handles all message create events to the bot, to pass them to child functions
func (c *CommandHandler) HandleMessageCreate(s *discordgo.Session, event *discordgo.MessageCreate) {
	if event.Author.ID == s.State.User.ID || event.Author.Bot {
		return
	}

	prefix, ok := checkMessagePrefix(s.State.User.ID, event)
	if !ok {
		return
	}

	prefixRemoved := strings.Fields(event.Content[len(prefix):])
	if len(prefixRemoved) < 1 {
		return
	}

	messageTrigger := strings.ToLower(prefixRemoved[0])
	cmd, ok := c.cmdMap[messageTrigger]
	if !ok {
		return
	}

	commandTokens := prefixRemoved[1:]

	data := &Data{
		Session:    s,
		GuildID:    event.GuildID,
		ChannelID:  event.ChannelID,
		Author:     event.Author,
		ParsedArgs: nil,
		Handler:    c,
		Message:    event.Message,
	}

	go runCommand(cmd, data, commandTokens)
}

// checkMessagePrefix checks a given content for the prefix or bot mention of the guild
func checkMessagePrefix(botID string, event *discordgo.MessageCreate) (prefix string, found bool) {
	prefix, ok := findBasicPrefix(event.Content, event.GuildID)
	if ok {
		return prefix, true
	}
	prefix, ok = findMentionPrefix(botID, event.Content)
	if ok {
		return prefix, true
	}

	return "", false
}

// findBasicPrefix finds a text based prefix such as "-" or "~"
func findBasicPrefix(message string, guildID string) (string, bool) {
	prefix := prfx.GuildPrefix(guildID)
	if strings.HasPrefix(message, prefix) {
		return prefix, true
	}

	return "", false
}

// findMentionPrefix finds a bot mention prefix such as @Summit
func findMentionPrefix(botID string, message string) (string, bool) {
	prefix := ""
	ok := false

	if strings.Index(message, "<@"+botID+">") == 0 {
		prefix = "<@" + botID + ">"
		ok = true
	} else if strings.Index(message, "<@!"+botID+">") == 0 {
		prefix = "<@!" + botID + ">"
		ok = true
	}

	return prefix, ok
}

// runCommand logs the command called by the bot, ensures the correct number of args is present, parses the args, then runs the command
func runCommand(cmd SummitCommand, data *Data, tokens []string) {
	logrus.WithFields(logrus.Fields{
		"Guild":           data.GuildID,
		"Command":         cmd.Command,
		"Triggering user": data.Author.ID},
	).Infoln("Executed command")

	err, embed := handleMissingOrInvalidArgs(cmd, data, tokens)
	if err != nil {
		functions.SendMessage(data.ChannelID, &discordgo.MessageSend{Embed: embed})
		return
	}

	cmd.Run(data)
}
