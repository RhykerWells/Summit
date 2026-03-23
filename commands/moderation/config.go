package moderation

import (
	"context"

	"github.com/RhykerWells/Summit/commands/moderation/models"
	"github.com/RhykerWells/Summit/common"
	"github.com/aarondl/sqlboiler/v4/boil"
)

// Config defines the general struct to pass data to and from the dashboard template/context data
type Config struct {
	// General
	GuildID                           string
	ModerationEnabled                 bool
	ModerationTriggerDeletionEnabled  bool
	ModerationTriggerDeletionSeconds  int64
	ModerationResponseDeletionEnabled bool
	ModerationResponseDeletionSeconds int64
	ModerationLogChannel              string

	// Warn
	WarnRequiredRoles []string

	// Mutes/Unmute
	MuteRequiredRoles []string
	MuteRole          string
	MuteManageRole    bool
	MuteUpdateRoles   []string

	// Kick
	KickRequiredRoles []string

	// Bans/Unbans
	BanRequiredRoles []string

	LastCaseID int64
}

// ConfigToSQLModel converts a Config struct to the relevant SQLBoiler model
func (c *Config) ConfigToSQLModel() *models.ModerationConfig {
	return &models.ModerationConfig{
		// General
		GuildID:                           c.GuildID,
		ModerationEnabled:                 c.ModerationEnabled,
		ModerationTriggerDeletionEnabled:  c.ModerationTriggerDeletionEnabled,
		ModerationTriggerDeletionSeconds:  c.ModerationTriggerDeletionSeconds,
		ModerationResponseDeletionEnabled: c.ModerationResponseDeletionEnabled,
		ModerationResponseDeletionSeconds: c.ModerationResponseDeletionSeconds,
		ModerationLogChannel:              c.ModerationLogChannel,

		/* Warn */
		WarnRequiredRoles: c.WarnRequiredRoles,

		/* Mute/Unmute */
		MuteRequiredRoles: c.MuteRequiredRoles,
		MuteRole:          c.MuteRole,
		MuteManageRole:    c.MuteManageRole,
		MuteUpdateRoles:   c.MuteUpdateRoles,

		/* Kick */
		KickRequiredRoles: c.KickRequiredRoles,

		/* Ban/Unban */
		BanRequiredRoles: c.BanRequiredRoles,

		LastCaseID: c.LastCaseID,
	}
}

// ConfigFromModel converts the guild config SQLBoiler model to a Config struct
func ConfigFromModel(m *models.ModerationConfig) *Config {
	return &Config{
		GuildID:                           m.GuildID,
		ModerationEnabled:                 m.ModerationEnabled,
		ModerationTriggerDeletionEnabled:  m.ModerationTriggerDeletionEnabled,
		ModerationTriggerDeletionSeconds:  m.ModerationTriggerDeletionSeconds,
		ModerationResponseDeletionEnabled: m.ModerationResponseDeletionEnabled,
		ModerationResponseDeletionSeconds: m.ModerationResponseDeletionSeconds,

		ModerationLogChannel: m.ModerationLogChannel,

		/* Warn */
		WarnRequiredRoles: m.WarnRequiredRoles,

		/* Mute */
		MuteRequiredRoles: m.MuteRequiredRoles,
		MuteRole:          m.MuteRole,
		MuteManageRole:    m.MuteManageRole,
		MuteUpdateRoles:   m.MuteUpdateRoles,

		/* Kick */
		KickRequiredRoles: m.KickRequiredRoles,

		/* Ban */
		BanRequiredRoles: m.BanRequiredRoles,

		LastCaseID: m.LastCaseID,
	}
}

// GetConfig returns the current or default guild config as a Config struct
func GetConfig(guildID string) *Config {
	model, err := models.FindModerationConfigG(context.Background(), guildID)
	if err == nil {
		return ConfigFromModel(model)
	}

	return &Config{
		GuildID: guildID,
	}
}

// SaveConfig saves the passed Config struct via SQLBoiler
func SaveConfig(config *Config) error {
	err := config.ConfigToSQLModel().UpsertG(context.Background(), true, []string{models.ModerationConfigColumns.GuildID}, boil.Infer(), boil.Infer())
	if err != nil {
		return err
	}

	if config.MuteManageRole && config.MuteRole != "" {
		RefreshMuteSettings(config)
	}

	return nil
}

// getGuildCases returns the guild cases
func getGuildCases(guildID string) models.ModerationCaseSlice {
	models, err := models.ModerationCases(models.ModerationCaseWhere.GuildID.EQ(guildID)).All(context.Background(), common.PQ)
	if err != nil {
		return nil
	}

	return models
}
