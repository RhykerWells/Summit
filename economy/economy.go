package economy

//go:generate sqlboiler --no-hooks psql

import (
	"context"

	eventsv2 "github.com/RhykerWells/Summit/bot/eventsV2"
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/economy/models"
	"github.com/RhykerWells/dispatch"
	"github.com/aarondl/sqlboiler/v4/boil"
)

func RegisterPlugin() {
	common.RegisterPlugin(&Plugin{})

	common.InitSchema("Economy", GuildEconomySchema...)
}

type Plugin struct{}

func (p *Plugin) PluginInfo() *common.PluginInfo {
	return &common.PluginInfo{
		Name:     "Economy",
		Category: &dispatch.CommandCategory{},
	}
}

func (p *Plugin) InitCommands(cmdHandler *dispatch.CommandHandler) {
	command.RegisterCommands(informationCommands...)
	command.RegisterCommands(incomeCommands...)
	command.RegisterCommands(transferCommands...)
	command.RegisterCommands(shopCommands...)
	command.RegisterCommands(inventoryCommands...)
}

func (p *Plugin) InitWeb() {
	initWeb()
}

func (p *Plugin) InitBot() {
	initEvents()

	common.Session.AddHandler(leaderboardPagination)
	common.Session.AddHandler(shopPagination)
	common.Session.AddHandler(inventoryPagination)
}

func initEvents() {
	eventsv2.AddHandler(handleGuildJoin, eventsv2.EventGuildCreate)
	eventsv2.AddHandler(handleGuildDelete, eventsv2.EventGuildDelete)
	eventsv2.AddHandler(handleGuildMemberAdd, eventsv2.EventGuildMemberAdd)
	eventsv2.AddHandler(handleGuildMemberRemove, eventsv2.EventGuildMemberRemove)
}

// handleGuildJoin creates the intial configs for the economy system for a specified guild
func handleGuildJoin(data *eventsv2.EventData) error {
	g := data.GuildCreate()

	config := GetConfig(g.ID)
	SaveConfig(config)

	return nil
}

// handleGuildDelete deletes the configs for the economy system for a specified guild
func handleGuildDelete(data *eventsv2.EventData) error {
	g := data.GuildDelete()

	config, err := models.EconomyConfigs(models.EconomyConfigWhere.GuildID.EQ(g.ID)).One(context.Background(), common.PQ)
	if err != nil {
		return err
	}

	_, err = config.Delete(context.Background(), common.PQ)
	if err != nil {
		return err
	}

	return nil
}

// handleGuildMemberAdd adds a member to the economy system
func handleGuildMemberAdd(data *eventsv2.EventData) error {
	m := data.GuildMemberAdd()

	guildID := m.GuildID

	config := GetConfig(guildID)
	userEntry := models.EconomyUser{
		GuildID: config.GuildID,
		UserID:  m.User.ID,
		Cash:    config.EconomyStartBalance,
		Bank:    0,
	}
	userEntry.Insert(context.Background(), common.PQ, boil.Infer())

	return nil
}

// handleGuildMemberRemove removes a guild member from the economy system
func handleGuildMemberRemove(data *eventsv2.EventData) error {
	m := data.GuildMemberRemove()

	models.EconomyUsers(models.EconomyUserWhere.GuildID.EQ(m.GuildID), models.EconomyUserWhere.UserID.EQ(m.User.ID)).DeleteAll(context.Background(), common.PQ)
	models.EconomyCooldowns(models.EconomyCooldownWhere.GuildID.EQ(m.GuildID), models.EconomyCooldownWhere.UserID.EQ(m.User.ID)).DeleteAll(context.Background(), common.PQ)
	models.EconomyUserInventories(models.EconomyUserInventoryWhere.GuildID.EQ(m.GuildID), models.EconomyUserInventoryWhere.UserID.EQ(m.User.ID)).DeleteAll(context.Background(), common.PQ)

	return nil
}
