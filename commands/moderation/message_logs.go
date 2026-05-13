package moderation

import (
	"context"
	"fmt"
	"strings"

	"github.com/RhykerWells/Summit/commands/moderation/models"
	"github.com/RhykerWells/Summit/common"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/bwmarrin/discordgo"
)

func createMessageLog(config *Config, channel *discordgo.Channel, author *discordgo.User, caseID int64) {
	if channel == nil {
		return
	}

	if author == nil {
		author = &discordgo.User{ID: "", Username: "Unknown"}
	}

	messages := MessageStore.GetMessages(channel.ID)

	tx, err := common.PQ.Begin()
	if err != nil {
		return
	}

	log := &models.ModerationMessageLog{
		GuildID:        config.GuildID,
		CaseID:         caseID,
		ChannelID:      channel.ID,
		ChannelName:    channel.Name,
		AuthorID:       author.ID,
		AuthorUsername: author.Username,
	}

	err = log.Insert(context.Background(), tx, boil.Infer())
	if err != nil {
		tx.Rollback()
		return
	}

	var logMessages []*models.ModerationMessageLogsMessage
	for _, msg := range messages {
		msgContent := msg.Content
		for _, attachment := range msg.Attachments {
			msgContent += fmt.Sprintf("\n[Attachment: %s]", attachment.URL)
		}

		msgContent = strings.ReplaceAll(msgContent, string(rune(0)), "")

		authorID := ""
		authorUsername := "Unknown"
		if msg.Author != nil {
			authorID = msg.Author.ID
			authorUsername = msg.Author.Username
		}

		logMessages = append(logMessages, &models.ModerationMessageLogsMessage{
			LogID:   log.ID,
			GuildID: config.GuildID,

			AuthorID:       authorID,
			AuthorUsername: authorUsername,

			Content: msgContent,

			CreatedAt: msg.CreatedAt,
			IsDeleted: msg.Deleted,
		})
	}

	err = log.AddLogModerationMessageLogsMessages(context.Background(), tx, true, logMessages...)
	if err != nil {
		tx.Rollback()
		return
	}

	err = tx.Commit()
	if err != nil {
		return
	}
}

// getGuildMessageLogs returns the guild message logs
func getGuildMessageLogs(guildID string) models.ModerationMessageLogSlice {
	models, err := models.ModerationMessageLogs(models.ModerationMessageLogWhere.GuildID.EQ(guildID)).All(context.Background(), common.PQ)
	if err != nil {
		return nil
	}

	return models
}

// getMessageLogByID returns the guild message log messages
func getMessageLogByID(logID int64) (*models.ModerationMessageLog, error) {
	log, err := models.FindModerationMessageLogG(context.Background(), logID)
	if err != nil {
		return nil, err
	}

	return log, nil
}
