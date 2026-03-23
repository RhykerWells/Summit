package core

import (
	"context"

	"github.com/RhykerWells/Summit/bot/core/models"
	"github.com/RhykerWells/Summit/common"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/bwmarrin/discordgo"
)

// Config defines the general struct to pass data to and from the dashboard template/context data
type Config struct {
	// General
	GuildID     string
	GuildPrefix string
}

// ConfigToSQLModel converts a Config struct to the relevant SQLBoiler model
func (c *Config) ConfigToSQLModel() *models.CoreConfig {
	return &models.CoreConfig{
		GuildID:     c.GuildID,
		GuildPrefix: c.GuildPrefix,
	}
}

// ConfigFromModel converts the guild config SQLBoiler model to a Config struct
func ConfigFromModel(m *models.CoreConfig) *Config {
	return &Config{
		GuildID:     m.GuildID,
		GuildPrefix: m.GuildPrefix,
	}
}

// GetConfig returns the current or default guild config as a Config struct
func GetConfig(guildID string) *Config {
	model, err := models.FindCoreConfigG(context.Background(), guildID)
	if err == nil {
		return ConfigFromModel(model)
	}

	return &Config{
		GuildID:     guildID,
		GuildPrefix: "~",
	}
}

// SaveConfig saves the passed Config struct via SQLBoiler
func SaveConfig(config *Config) error {
	err := config.ConfigToSQLModel().UpsertG(context.Background(), true, []string{models.CoreConfigColumns.GuildID}, boil.Infer(), boil.Infer())
	if err != nil {
		return err
	}

	return nil
}

// DeleteConfig deletes the passed Config struct via SQLBoiler
func DeleteConfig(config *Config) error {
	_, err := config.ConfigToSQLModel().Delete(context.Background(), common.PQ)
	if err != nil {
		return err
	}

	return nil
}

// guildAddCoreConfig adds and saves the base moderation config when added to a new guild
func guildAddCoreConfig(g *discordgo.GuildCreate) {
	config := GetConfig(g.ID)
	SaveConfig(config)
}

// guildDeleteCoreConfig removes the base moderation config when removed from a guild
func guildDeleteCoreConfig(g *discordgo.GuildDelete) {
	config := GetConfig(g.ID)
	DeleteConfig(config)
}
