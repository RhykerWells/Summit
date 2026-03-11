package economy

//go:generate sqlboiler --no-hooks psql

import (
	"context"

	"github.com/RhykerWells/Summit/bot/events"
	"github.com/RhykerWells/Summit/commands/economy/models"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/dispatch"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/bwmarrin/discordgo"
)

// EconomySetup runs the following:
//   - The schema initialiser
//   - Registration of the guild and user join/leave functions
//   - Initialises the web plugin
//   - Registration of the economy commands & their pagination
func EconomySetup(cmdHandler *dispatch.CommandHandler) {
	common.InitSchema("Economy", GuildEconomySchema...)
	events.RegisterGuildJoinfunctions([]func(g *discordgo.GuildCreate){
		guildAddEconomyConfig,
	})
	events.RegisterGuildLeavefunctions([]func(g *discordgo.GuildDelete){
		guildDeleteEconomyConfig,
	})
	events.RegisterGuildMemberJoinfunctions([]func(g *discordgo.GuildMemberAdd){
		guildMemberAddToEconomy,
	})
	events.RegisterGuildMemberLeavefunctions([]func(g *discordgo.GuildMemberRemove){
		guildMemberRemoveFromEconomy,
	})

	initWeb()

	// Information commands
	cmdHandler.RegisterCommands(informationCommands...)

	// Income commands
	cmdHandler.RegisterCommands(incomeCommands...)

	// Transfer commands
	cmdHandler.RegisterCommands(transferCommands...)

	// Shop commands
	cmdHandler.RegisterCommands(shopCommands...)

	// Inventory commands
	cmdHandler.RegisterCommands(inventoryCommands...)

	common.Session.AddHandler(leaderboardPagination)
	common.Session.AddHandler(shopPagination)
	common.Session.AddHandler(inventoryPagination)
}

// guildAddEconomyConfig creates the intial configs for the economy system for a specified guild
func guildAddEconomyConfig(g *discordgo.GuildCreate) {
	config := GetConfig(g.ID)
	SaveConfig(config)
}

// guildAddEconomyConfig deletes the configs for the economy system for a specified guild
func guildDeleteEconomyConfig(g *discordgo.GuildDelete) {
	config, err := models.EconomyConfigs(models.EconomyConfigWhere.GuildID.EQ(g.ID)).One(context.Background(), common.PQ)
	if err != nil {
		return
	}

	config.Delete(context.Background(), common.PQ)
}

// guildMemberAddToEconomy adds a member to the economy system
func guildMemberAddToEconomy(m *discordgo.GuildMemberAdd) {
	config := GetConfig(m.GuildID)
	userEntry := models.EconomyUser{
		GuildID: config.GuildID,
		UserID:  m.User.ID,
		Cash:    config.EconomyStartBalance,
		Bank:    0,
	}
	userEntry.Insert(context.Background(), common.PQ, boil.Infer())
}

// guildMemberRemoveFromEconomy removes a guild member from the economy system
func guildMemberRemoveFromEconomy(m *discordgo.GuildMemberRemove) {
	models.EconomyUsers(models.EconomyUserWhere.GuildID.EQ(m.GuildID), models.EconomyUserWhere.UserID.EQ(m.User.ID)).DeleteAll(context.Background(), common.PQ)
	models.EconomyCooldowns(models.EconomyCooldownWhere.GuildID.EQ(m.GuildID), models.EconomyCooldownWhere.UserID.EQ(m.User.ID)).DeleteAll(context.Background(), common.PQ)
	models.EconomyUserInventories(models.EconomyUserInventoryWhere.GuildID.EQ(m.GuildID), models.EconomyUserInventoryWhere.UserID.EQ(m.User.ID)).DeleteAll(context.Background(), common.PQ)
}
