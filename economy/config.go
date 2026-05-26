package economy

import (
	"context"

	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/economy/models"
	"github.com/aarondl/sqlboiler/v4/boil"
)

// Config defines the general struct to pass data to and from the dashboard template/context data
type Config struct {
	// General
	GuildID             string
	EconomyEnabled      bool
	EconomySymbol       string
	EconomyStartBalance int64

	// Money making management
	EconomyMinReturn int64
	EconomyMaxReturn int64
	EconomyMaxBet    int64

	// Custom responses
	EconomyCustomWorkResponsesEnabled  bool
	EconomyCustomCrimeResponsesEnabled bool
}

// ConfigToSQLModel converts a Config struct to the relevant SQLBoiler model
func (c *Config) ConfigToSQLModel() *models.EconomyConfig {
	return &models.EconomyConfig{
		// General
		GuildID:             c.GuildID,
		EconomyEnabled:      c.EconomyEnabled,
		EconomySymbol:       c.EconomySymbol,
		EconomyStartBalance: c.EconomyStartBalance,

		// Money making management
		EconomyMinReturn: c.EconomyMinReturn,
		EconomyMaxReturn: c.EconomyMaxReturn,
		EconomyMaxBet:    c.EconomyMaxBet,

		// Custom responses
		EconomyCustomWorkResponsesEnabled:  c.EconomyCustomWorkResponsesEnabled,
		EconomyCustomCrimeResponsesEnabled: c.EconomyCustomCrimeResponsesEnabled,
	}
}

// ConfigFromModel converts the guild config SQLBoiler model to a Config struct
func ConfigFromModel(m *models.EconomyConfig) *Config {
	return &Config{
		// General
		GuildID:             m.GuildID,
		EconomyEnabled:      m.EconomyEnabled,
		EconomySymbol:       m.EconomySymbol,
		EconomyStartBalance: m.EconomyStartBalance,

		// Money making management
		EconomyMinReturn: m.EconomyMinReturn,
		EconomyMaxReturn: m.EconomyMaxReturn,
		EconomyMaxBet:    m.EconomyMaxBet,

		// Custom responses
		EconomyCustomWorkResponsesEnabled:  m.EconomyCustomWorkResponsesEnabled,
		EconomyCustomCrimeResponsesEnabled: m.EconomyCustomCrimeResponsesEnabled,
	}
}

// GetConfig returns the current or default guild config as a Config struct
func GetConfig(guildID string) *Config {
	model, err := models.FindEconomyConfigG(context.Background(), guildID)
	if err == nil {
		return ConfigFromModel(model)
	}

	defaultConfig := &Config{
		GuildID:                            guildID,
		EconomyEnabled:                     false,
		EconomySymbol:                      "£",
		EconomyStartBalance:                200,
		EconomyMinReturn:                   200,
		EconomyMaxReturn:                   500,
		EconomyMaxBet:                      5000,
		EconomyCustomWorkResponsesEnabled:  false,
		EconomyCustomCrimeResponsesEnabled: false,
	}
	defaultConfig.ConfigToSQLModel().InsertG(context.Background(), boil.Infer())

	return defaultConfig
}

// SaveConfig saves the passed Config struct via SQLBoiler
func SaveConfig(config *Config) error {
	err := config.ConfigToSQLModel().UpsertG(context.Background(), true, []string{models.EconomyConfigColumns.GuildID}, boil.Infer(), boil.Infer())
	if err != nil {
		return err
	}

	return nil
}

// getGuildShop returns the guild shop
func getGuildShop(guildID string) models.EconomyShopSlice {
	models, err := models.EconomyShops(models.EconomyShopWhere.GuildID.EQ(guildID)).All(context.Background(), common.PQ)
	if err != nil {
		return nil
	}

	return models
}
