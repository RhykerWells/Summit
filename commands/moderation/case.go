package moderation

import (
	"context"
	"fmt"
	"time"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/commands/moderation/models"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/durationutil"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
)

// logAction defines the structure of each moderation action and the properties used within the log
type logAction struct {
	CaseType string
	Name     string
	Colour   int
}

var (
	logWarn   = logAction{Name: "Warned", CaseType: "Warn", Colour: 0xFCA253}
	logMute   = logAction{Name: "Muted", CaseType: "Mute", Colour: 0x5772BE}
	logUnmute = logAction{Name: "Unmuted", CaseType: "Unmute", Colour: common.SuccessGreen}
	logKick   = logAction{Name: "Kicked", CaseType: "Kick", Colour: 0xF2A013}
	logBan    = logAction{Name: "Banned", CaseType: "Ban", Colour: 0xD64848}
	logUnban  = logAction{Name: "Unbanned", CaseType: "Unban", Colour: common.SuccessGreen}
)

const (
	caseEmoji    string = "<:ID:1369739780958457966>"
	userEmoji    string = "<:Member:1369740929568739499>"
	actionEmoji  string = "<:Action:1369745870001799321>"
	channelEmoji string = "<:Channel:1369743815887294687>"
	reasonEmoji  string = "<:Reason:1369744280310124624>"
)

// getNewCaseID returns the next case ID
func getNewCaseID(config *Config) int64 {
	return config.LastCaseID + 1
}

// incrementCaseID increments the case ID within the guild config
func incrementCaseID(config *Config) error {
	config.LastCaseID++
	err := SaveConfig(config)

	return err
}

// removeFailedCase removes a case from the database
func removeFailedCase(caseData models.ModerationCase) {
	caseData.Delete(context.Background(), common.PQ)
}

// buildLogEmbed constructs and returns the embed object to be sent within the case log
func buildLogEmbed(caseNumber int64, author, target *discordgo.User, action logAction, channelID, reason string, duration ...time.Duration) *discordgo.MessageEmbed {
	humanReadableCaseNumber := humanize.Comma(caseNumber)

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    fmt.Sprintf("%s (ID %s)", author.Username, author.ID),
			IconURL: author.AvatarURL("1024"),
		},
		Description: fmt.Sprintf("%s **Case number:** %s\n%s **Who:** %s `(ID %s)`\n%s **Action:** %s\n%s **Channel:** <#%s>\n%s **Reason:** %s", caseEmoji, humanReadableCaseNumber, userEmoji, target.Mention(), target.ID, actionEmoji, action.Name, channelEmoji, channelID, reasonEmoji, reason),
		Color:       action.Colour,
	}

	if action != logMute && action != logBan {
		return embed
	}

	if len(duration) > 0 {
		d := duration[0]
		if d > 0 {
			embed.Footer = &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("Duration: %s", durationutil.HumanizeDuration(d)),
			}
		} else {
			embed.Footer = &discordgo.MessageEmbedFooter{
				Text: "Duration: Indefinite",
			}
		}
	}

	return embed
}

// generateLogLink returns the log link of the case within discord
func generateLogLink(guildID, channelID, messageID string) string {
	return fmt.Sprintf("https://discord.com/channels/%s/%s/%s", guildID, channelID, messageID)
}

// createCase generates the moderationc case for the database and saves it, then attempts to build and log the Discord case log, if this fails the case will be removed
func createCase(config *Config, author, target *discordgo.Member, action logAction, channelID, reason string, duration ...time.Duration) error {
	caseID := getNewCaseID(config)

	caseData := models.ModerationCase{
		CaseID:           caseID,
		GuildID:          config.GuildID,
		StaffUsername:    author.User.Username,
		StaffID:          author.User.ID,
		OffenderUsername: target.User.Username,
		OffenderID:       target.User.ID,
		Reason:           reason,
		Action:           action.CaseType,
		LogLink:          "",
	}
	if err := caseData.Insert(context.Background(), common.PQ, boil.Infer()); err != nil {
		return err
	}

	embed := buildLogEmbed(caseID, author.User, target.User, action, channelID, reason, duration...)
	msg, err := functions.SendMessage(config.ModerationLogChannel, &discordgo.MessageSend{Embed: embed})
	if err != nil {
		removeFailedCase(caseData)
		return err
	}

	caseData.LogLink = generateLogLink(config.GuildID, config.ModerationLogChannel, msg.ID)
	if _, err := caseData.Update(context.Background(), common.PQ, boil.Infer()); err != nil {
		functions.DeleteMessage(config.ModerationLogChannel, msg.ID)
		removeFailedCase(caseData)
		return err
	}

	if err := incrementCaseID(config); err != nil {
		functions.DeleteMessage(config.ModerationLogChannel, msg.ID)
		removeFailedCase(caseData)
		return err
	}

	return nil
}

func buildDMEmbed(config *Config, target *discordgo.User, action logAction, reason string, duration ...time.Duration) *discordgo.MessageEmbed {
	embedDuration := ""
	if len(duration) > 0 {
		d := duration[0]
		embedDuration = fmt.Sprintf("\n\n**Duration:** %s", durationutil.HumanizeDuration(d))
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    fmt.Sprintf("%s (ID %s)", target.Username, target.ID),
			IconURL: target.AvatarURL("1024"),
		},
		Description: fmt.Sprintf("**Server:** %s\n\n**Action:** %s%s\n\n**Reason:** %s", config.GuildID, action.CaseType, embedDuration, reason),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	return embed
}
