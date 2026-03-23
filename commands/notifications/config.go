package notifications

import (
	"context"

	"github.com/RhykerWells/Summit/commands/notifications/models"
	"github.com/aarondl/sqlboiler/v4/boil"
)

// Config defines the general struct to pass data to and from the dashboard template/context data
type Config struct {
	GuildID string

	JoinServerChannel string `valid:"channel,allowEmpty"`
	JoinServerMessage string `valid:",allowEmpty"`

	LeaveServerChannel string `valid:"channel,allowEmpty"`
	LeaveServerMessage string `valid:",allowEmpty"`
}

// ConfigToSQLModel converts a Config struct to the relevant SQLBoiler model
func (c *Config) ConfigToSQLModel() *models.NotificationsConfig {
	return &models.NotificationsConfig{
		GuildID: c.GuildID,

		JoinServerChannel: c.JoinServerChannel,
		JoinServerMessage: c.JoinServerMessage,

		LeaveServerChannel: c.LeaveServerChannel,
		LeaveServerMessage: c.LeaveServerMessage,
	}
}

// ConfigFromModel converts the guild config SQLBoiler model to a Config struct
func ConfigFromModel(m *models.NotificationsConfig) *Config {
	return &Config{
		GuildID: m.GuildID,

		JoinServerChannel: m.JoinServerChannel,
		JoinServerMessage: m.JoinServerMessage,

		LeaveServerChannel: m.LeaveServerChannel,
		LeaveServerMessage: m.LeaveServerMessage,
	}
}

// GetConfig returns the current or default guild config as a Config struct
func GetConfig(guildID string) *Config {
	model, err := models.FindNotificationsConfigG(context.Background(), guildID)
	if err == nil {
		return ConfigFromModel(model)
	}

	return &Config{
		GuildID: guildID,
	}
}

// SaveConfig saves the passed Config struct via SQLBoiler
func SaveConfig(config *Config) error {
	err := config.ConfigToSQLModel().UpsertG(context.Background(), true, []string{models.NotificationsConfigColumns.GuildID}, boil.Infer(), boil.Infer())
	if err != nil {
		return err
	}

	return nil
}
