package economy

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/economy/models"
	"github.com/RhykerWells/dispatch"
	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
)

type FullEconomyMember struct {
	EconomyUser          models.EconomyUser
	EconomyUserInventory models.EconomyUserInventorySlice
}

// getFullEconomyMember returns a members full economy information
func getFullEconomyMember(config *Config, memberID string) FullEconomyMember {
	var economyUser *models.EconomyUser
	economyUser, err := models.EconomyUsers(models.EconomyUserWhere.GuildID.EQ(config.GuildID), models.EconomyUserWhere.UserID.EQ(memberID)).One(context.Background(), common.PQ)
	if err != nil {
		economyUser = &models.EconomyUser{
			GuildID:     config.GuildID,
			UserID:      memberID,
			Cash:        0,
			Bank:        0,
			Cfwinchance: 50,
		}
	}

	var economyUserInventory models.EconomyUserInventorySlice
	economyUserInventory, err = models.EconomyUserInventories(models.EconomyUserInventoryWhere.GuildID.EQ(config.GuildID), models.EconomyUserInventoryWhere.UserID.EQ(memberID)).All(context.Background(), common.PQ)
	if err != nil {
		economyUserInventory = models.EconomyUserInventorySlice{}
	}

	return FullEconomyMember{
		EconomyUser:          *economyUser,
		EconomyUserInventory: economyUserInventory,
	}
}

func getUserCashRank(guildID, userID string) (int64, string) {
	var position int64
	err := common.PQ.QueryRowContext(context.Background(), `SELECT position FROM (SELECT user_id, RANK() OVER (ORDER BY cash DESC) AS position FROM economy_cash WHERE guild_id=$1) AS s WHERE user_id=$2`, guildID, userID).Scan(&position)
	if err != nil {
		return position, ""
	}

	ordinal := "th"
	cent := functions.ToInt64(math.Mod(float64(position), float64(100)))
	dec := functions.ToInt64(math.Mod(float64(position), float64(10)))
	if cent < 10 || cent > 19 {
		switch dec {
		case 1:
			ordinal = "st"
		case 2:
			ordinal = "nd"
		case 3:
			ordinal = "rd"
		}
	}
	ordinalPosition := fmt.Sprintf("%d%s.", position, ordinal)

	return position, ordinalPosition
}

func getPageNumber(pageNum int) int {
	if pageNum < 1 {
		pageNum = 1
	}

	return pageNum
}

func sendPaginatedEmbed[T any](channelID string, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent, items []T, page int) {
	embed.Footer = &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Page: %d", page)}

	if page != 1 {
		row := components[0].(discordgo.ActionsRow)
		btnPrev := row.Components[0].(discordgo.Button)
		btnPrev.Disabled = false
		row.Components[0] = btnPrev
		components[0] = row
	}
	if len(items) > (page-1)*10 {
		row := components[0].(discordgo.ActionsRow)
		btnNext := row.Components[1].(discordgo.Button)
		btnNext.Disabled = false
		row.Components[1] = btnNext
		components[0] = row
	}

	msg, _ := functions.SendMessage(channelID, &discordgo.MessageSend{Embed: embed, Components: components})
	go disableButtons(channelID, msg.ID)
}

func disableButtons(channelID, messageID string) {
	time.Sleep(10 * time.Second)

	message, _ := common.Session.ChannelMessage(channelID, messageID)
	components := message.Components
	row := components[0].(*discordgo.ActionsRow)
	btnPrev := row.Components[0].(*discordgo.Button)
	btnNext := row.Components[1].(*discordgo.Button)
	btnPrev.Disabled = true
	btnNext.Disabled = true
	row.Components[0] = btnPrev
	row.Components[1] = btnNext
	components[0] = row

	updatedMessage := &discordgo.MessageSend{
		Embed:      message.Embeds[0],
		Components: components,
	}
	functions.EditMessage(channelID, messageID, updatedMessage)
}

// commandCooldown returns a boolean whether the current command is on a cooldown for the user
func commandCooldown(config *Config, userID string, cooldownType string) (bool, int64) {
	cooldown, err := models.EconomyCooldowns(models.EconomyCooldownWhere.GuildID.EQ(config.GuildID), models.EconomyCooldownWhere.UserID.EQ(userID), models.EconomyCooldownWhere.Type.EQ(cooldownType)).One(context.Background(), common.PQ)
	if err != nil {
		return false, 0
	}

	if !cooldown.ExpiresAt.Time.After(time.Now()) {
		return false, 0
	}

	return true, time.Now().Unix() + int64(time.Until(cooldown.ExpiresAt.Time).Seconds())
}

func betAmount(config *Config, economyMember FullEconomyMember, amount any) int64 {
	switch amount {
	case "all":
		return economyMember.EconomyUser.Cash
	case "max":
		if config.EconomyMaxBet > 0 {
			return config.EconomyMaxBet
		} else {
			return economyMember.EconomyUser.Cash
		}
	default:
		return functions.ToInt64(amount)
	}
}

var (
	activeGames = make(map[string]*RouletteGame)
)

type RouletteGame struct {
	Bet       int64
	PlayerIDs []string
	HostID    string
	IsActive  bool
}

var informationCommands = []*dispatch.Command{
	{
		Command:     "balance",
		Category:    command.CategoryEconomy,
		Aliases:     []string{"bal"},
		Description: "Views your balance in the economy",
		Run: (func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)

			fullEconomyMember := getFullEconomyMember(guildConfig, data.Author.ID)
			var cash, bank int64 = fullEconomyMember.EconomyUser.Cash, fullEconomyMember.EconomyUser.Bank
			_, ordinalPosition := getUserCashRank(data.Guild.ID, data.Author.ID)

			embed := &discordgo.MessageEmbed{
				Author: &discordgo.MessageEmbedAuthor{
					Name:    data.Author.Username,
					IconURL: data.Author.AvatarURL("256"),
				},
				Description: fmt.Sprintf("%s's balance\nLeaderboard rank %s", data.Author.Mention(), ordinalPosition),
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Cash",
						Value:  fmt.Sprint(cash),
						Inline: true,
					},
					{
						Name:   "Bank",
						Value:  fmt.Sprint(bank),
						Inline: true,
					},
					{
						Name:   "Networth",
						Value:  fmt.Sprint(cash + bank),
						Inline: true,
					},
				},
				Timestamp: time.Now().Format(time.RFC3339),
				Color:     common.SuccessGreen,
			}

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
			return nil
		}),
	},
	{
		Command:  "leaderboard",
		Category: command.CategoryEconomy,
		Aliases:  []string{"lb", "top"},
		Args: []*dispatch.Arg{
			{Name: "Page", Type: &dispatch.IntArg{Min: 1}},
		},
		Description: "Views your server leaderboard",
		Run: (func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)

			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Guild.Name + " leaderboard", IconURL: data.Guild.IconURL("256")}, Description: "No users are on this page", Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}
			components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "previous", Style: 4, Disabled: true, CustomID: "leaderboard_back"}, discordgo.Button{Label: "next", Style: 3, Disabled: true, CustomID: "leaderboard_forward"}}}}

			var page int = 1
			if len(data.ParsedArgs) > 0 {
				pageNumFromArg, _ := data.ParsedArgs[0].Value.(int)
				page = getPageNumber(pageNumFromArg)
			}

			offset := (page - 1) * 10
			economyUsers, err := models.EconomyUsers(models.EconomyUserWhere.GuildID.EQ(data.Guild.ID), qm.OrderBy("cash DESC"), qm.Offset(offset)).All(context.Background(), common.PQ)
			if err != nil || len(economyUsers) == 0 {
				sendPaginatedEmbed(data.Channel.ID, embed, components, economyUsers, page)
				return nil
			}

			embed.Description = ""
			embed.Color = common.SuccessGreen
			for i, economyUser := range economyUsers {
				if i == 10 {
					break
				}

				user, _ := functions.GetUser(economyUser.UserID)
				cash := humanize.Comma(economyUser.Cash)
				position, ordinalPosition := getUserCashRank(data.Guild.ID, economyUser.UserID)
				if medal, ok := map[int64]string{1: "🥇", 2: "🥈", 3: "🥉"}[position]; ok {
					ordinalPosition = medal
				}
				embed.Description += fmt.Sprintf("**%v** %s **•** %s%s\n", ordinalPosition, user.Username, guildConfig.EconomySymbol, cash)
			}

			sendPaginatedEmbed(data.Channel.ID, embed, components, economyUsers, page)
			return nil
		}),
	},
}

func leaderboardPagination(s *discordgo.Session, b *discordgo.InteractionCreate) {
	if b.MessageComponentData().CustomID != "leaderboard_back" && b.MessageComponentData().CustomID != "leaderboard_forward" {
		return
	}

	guildConfig := GetConfig(b.GuildID)
	guild, err := common.Session.Guild(b.GuildID)
	if err != nil || guild == nil {
		return
	}
	embed := []*discordgo.MessageEmbed{{Author: &discordgo.MessageEmbedAuthor{Name: guild.Name + " leaderboard", IconURL: guild.IconURL("256")}, Description: "No users are on this page", Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}}
	components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "previous", Style: 4, Disabled: true, CustomID: "leaderboard_back"}, discordgo.Button{Label: "next", Style: 3, Disabled: true, CustomID: "leaderboard_forward"}}}}

	re := regexp.MustCompile(`\d+`)
	page, _ := strconv.Atoi(re.FindString(b.Message.Embeds[0].Footer.Text))
	switch b.MessageComponentData().CustomID {
	case "leaderboard_forward":
		page++
	case "leaderboard_back":
		page--
	}

	offset := (page - 1) * 10
	economyUsers, err := models.EconomyUsers(models.EconomyUserWhere.GuildID.EQ(b.GuildID), qm.OrderBy("cash DESC"), qm.Offset(offset)).All(context.Background(), common.PQ)
	if err == nil && len(economyUsers) == 0 {
		embed[0].Description = ""
		embed[0].Color = common.SuccessGreen
	}

	rank := (page - 1) * 10
	for i, economyUser := range economyUsers {
		if i == 10 {
			break
		}

		user, _ := functions.GetUser(economyUser.UserID)
		cash := humanize.Comma(economyUser.Cash)
		position, ordinalPosition := getUserCashRank(b.GuildID, economyUser.UserID)
		if medal, ok := map[int64]string{1: "🥇", 2: "🥈", 3: "🥉"}[position]; ok {
			ordinalPosition = medal
		}

		embed[0].Description += fmt.Sprintf("**%v** %s **•** %s%s\n", ordinalPosition, user.Username, guildConfig.EconomySymbol, cash)
	}

	embed[0].Footer = &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Page: %d", page)}
	if page != 1 {
		row := components[0].(discordgo.ActionsRow)
		btnPrev := row.Components[0].(discordgo.Button)
		btnPrev.Disabled = false
		row.Components[0] = btnPrev
		components[0] = row
	}
	if len(economyUsers) > rank {
		row := components[0].(discordgo.ActionsRow)
		btnNext := row.Components[1].(discordgo.Button)
		btnNext.Disabled = false
		row.Components[1] = btnNext
		components[0] = row
	}

	common.Session.InteractionRespond(b.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Embeds: embed, Components: components}})
}

var incomeCommands = []*dispatch.Command{
	{
		Command:     "work",
		Category:    command.CategoryEconomy,
		Description: "Work work work",
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)
			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Author.Username, IconURL: data.Author.AvatarURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			cooldownActive, cooldown := commandCooldown(guildConfig, data.Author.ID, "work")
			if cooldownActive {
				return fmt.Errorf("This command is on cooldown for <t:%d:R>", cooldown)
			}

			fullEconomyMember := getFullEconomyMember(guildConfig, data.Author.ID)
			cash := fullEconomyMember.EconomyUser.Cash

			payout := rand.Int63n(guildConfig.EconomyMaxReturn-guildConfig.EconomyMinReturn+1) + guildConfig.EconomyMinReturn

			embed.Description = fmt.Sprintf("You decided to work today! You got paid a hefty %s%s", guildConfig.EconomySymbol, humanize.Comma(payout))
			responses, _ := models.EconomyResponses(models.EconomyResponseWhere.GuildID.EQ(guildConfig.GuildID), models.EconomyResponseWhere.Type.EQ("work")).All(context.Background(), common.PQ)
			if guildConfig.EconomyCustomWorkResponsesEnabled && len(responses) > 0 {
				embed.Description = strings.ReplaceAll(responses[rand.Intn(len(responses))].Response, "(amount)", fmt.Sprintf("%s%s", guildConfig.EconomySymbol, humanize.Comma(payout)))
			}

			cash = cash + payout
			embed.Color = common.SuccessGreen

			userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: data.Author.ID, Cash: cash}
			userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash), boil.Infer())
			cooldowns := models.EconomyCooldown{GuildID: data.Guild.ID, UserID: data.Author.ID, Type: "work", ExpiresAt: null.Time{Time: time.Now().Add(3600 * time.Second), Valid: true}}
			cooldowns.Upsert(context.Background(), common.PQ, true, []string{models.EconomyCooldownColumns.GuildID, models.EconomyCooldownColumns.UserID, models.EconomyCooldownColumns.Type}, boil.Whitelist(models.EconomyCooldownColumns.ExpiresAt), boil.Infer())

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
			return nil
		},
	},
	{
		Command:     "crime",
		Category:    command.CategoryEconomy,
		Description: "Pew pew pew",
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)
			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Author.Username, IconURL: data.Author.AvatarURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			cooldownActive, cooldown := commandCooldown(guildConfig, data.Author.ID, "crime")
			if cooldownActive {
				return fmt.Errorf("This command is on cooldown for <t:%d:R>", cooldown)
			}

			fullEconomyMember := getFullEconomyMember(guildConfig, data.Author.ID)
			cash := fullEconomyMember.EconomyUser.Cash

			payout := rand.Int63n(guildConfig.EconomyMaxReturn-guildConfig.EconomyMinReturn+1) + guildConfig.EconomyMinReturn

			if rand.Int63n(2) == 1 {
				embed.Description = fmt.Sprintf("You broke the law for a pretty penny! You made %s%s in your crime spree", guildConfig.EconomySymbol, humanize.Comma(payout))
				responses, _ := models.EconomyResponses(models.EconomyResponseWhere.GuildID.EQ(guildConfig.GuildID), models.EconomyResponseWhere.Type.EQ("crime")).All(context.Background(), common.PQ)
				if guildConfig.EconomyCustomCrimeResponsesEnabled && len(responses) > 0 {
					embed.Description = strings.ReplaceAll(responses[rand.Intn(len(responses))].Response, "(amount)", fmt.Sprintf("%s%s", guildConfig.EconomySymbol, humanize.Comma(payout)))
				}
				embed.Color = common.SuccessGreen
				cash = cash + payout
			} else {
				embed.Description = fmt.Sprintf("You broke the law and got caught! You were arrested and lost %s%s", guildConfig.EconomySymbol, humanize.Comma(payout))
				cash = cash - payout
			}

			userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: data.Author.ID, Cash: cash}
			userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash), boil.Infer())
			cooldowns := models.EconomyCooldown{GuildID: data.Guild.ID, UserID: data.Author.ID, Type: "crime", ExpiresAt: null.Time{Time: time.Now().Add(3600 * time.Second), Valid: true}}
			cooldowns.Upsert(context.Background(), common.PQ, true, []string{models.EconomyCooldownColumns.GuildID, models.EconomyCooldownColumns.UserID, models.EconomyCooldownColumns.Type}, boil.Whitelist(models.EconomyCooldownColumns.ExpiresAt), boil.Infer())

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
			return nil
		},
	},
	{
		Command:      "rob",
		Category:     command.CategoryEconomy,
		Aliases:      []string{"steal"},
		Description:  "Money money money money money",
		ArgsRequired: 1,
		Args: []*dispatch.Arg{
			{Name: "Member", Type: dispatch.Member},
		},
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)
			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Author.Username, IconURL: data.Author.AvatarURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			cooldownActive, cooldown := commandCooldown(guildConfig, data.Author.ID, "rob")
			if cooldownActive {
				return fmt.Errorf("This command is on cooldown for <t:%d:R>", cooldown)
			}

			fullEconomyMember := getFullEconomyMember(guildConfig, data.Author.ID)
			cash := fullEconomyMember.EconomyUser.Cash

			targetMember := data.ParsedArgs[0].Value.(*discordgo.Member)
			if targetMember.User.ID == data.Author.ID {
				return errors.New("You cannot rob yourself")
			}

			fullEconomyMemberTarget := getFullEconomyMember(guildConfig, targetMember.User.ID)
			targetCash := fullEconomyMemberTarget.EconomyUser.Cash

			if targetCash < 0 {
				return errors.New("This user has no cash to steal :(")
			}

			payout := rand.Int63n(targetCash) + 1
			embed.Description = fmt.Sprintf("You stole %s%s from %s", guildConfig.EconomySymbol, humanize.Comma(payout), targetMember.Mention())
			cash = cash + payout
			targetCash = targetCash - payout

			userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: data.Author.ID, Cash: cash}
			userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash), boil.Infer())
			targetEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: targetMember.User.ID, Cash: targetCash}
			targetEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash), boil.Infer())
			cooldowns := models.EconomyCooldown{GuildID: data.Guild.ID, UserID: data.Author.ID, Type: "rob", ExpiresAt: null.Time{Time: time.Now().Add(18000 * time.Second), Valid: true}}
			cooldowns.Upsert(context.Background(), common.PQ, true, []string{models.EconomyCooldownColumns.GuildID, models.EconomyCooldownColumns.UserID, models.EconomyCooldownColumns.Type}, boil.Whitelist(models.EconomyCooldownColumns.ExpiresAt), boil.Infer())

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
			return nil
		},
	},
	{
		Command:      "chickenfight",
		Category:     command.CategoryEconomy,
		Description:  "Chicken fight for a payout of <Bet> with a base payout of 50%. Increases each win up to 70%",
		ArgsRequired: 1,
		Args: []*dispatch.Arg{
			{Name: "Bet", Type: &command.BetArg{Min: 1}},
		},
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)
			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Author.Username, IconURL: data.Author.AvatarURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			cooldownActive, cooldown := commandCooldown(guildConfig, data.Author.ID, "chickenfight")
			if cooldownActive {
				return fmt.Errorf("This command is on cooldown for <t:%d:R>", cooldown)
			}

			fullEconomyMember := getFullEconomyMember(guildConfig, data.Author.ID)
			cash := fullEconomyMember.EconomyUser.Cash
			bet := betAmount(guildConfig, fullEconomyMember, data.ParsedArgs[0].Value)

			if bet > cash {
				return fmt.Errorf("You can't bet more than you have in your hand. You currently have %s%s", guildConfig.EconomySymbol, humanize.Comma(fullEconomyMember.EconomyUser.Cash))
			}
			if guildConfig.EconomyMaxBet > 0 && bet > guildConfig.EconomyMaxBet {
				return fmt.Errorf("You can't bet more than the servers limit. The limit is %s%s", guildConfig.EconomySymbol, humanize.Comma(guildConfig.EconomyMaxBet))
			}

			chicken, exists := models.EconomyUserInventories(models.EconomyUserInventoryWhere.GuildID.EQ(data.Guild.ID), models.EconomyUserInventoryWhere.UserID.EQ(data.Author.ID), models.EconomyUserInventoryWhere.Name.IN([]string{"Chicken", "chicken"})).One(context.Background(), common.PQ)
			if exists != nil {
				return errors.New("You don't have this item\nBuy it in the shop!")
			}

			winChance := fullEconomyMember.EconomyUser.Cfwinchance
			win := (rand.Int63n(100) + 1) <= winChance

			if win {
				if winChance != 70 {
					winChance = winChance + 1
				}
				cash = cash + bet
				embed.Description = "Your chicken won the fight! Play again with an increased win chance"
				embed.Color = common.SuccessGreen
			} else {
				winChance = 50
				cash = cash - bet
				embed.Description = "Your chicken lost the fight and died :(\nBuy a new one to play again"
				chicken.Delete(context.Background(), common.PQ)
			}

			userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: data.Author.ID, Cash: cash, Cfwinchance: winChance}
			userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash, models.EconomyUserColumns.Cfwinchance), boil.Infer())
			cooldowns := models.EconomyCooldown{GuildID: data.Guild.ID, UserID: data.Author.ID, Type: "chickenfight", ExpiresAt: null.Time{Time: time.Now().Add(300 * time.Second), Valid: true}}
			cooldowns.Upsert(context.Background(), common.PQ, true, []string{models.EconomyCooldownColumns.GuildID, models.EconomyCooldownColumns.UserID, models.EconomyCooldownColumns.Type}, boil.Whitelist(models.EconomyCooldownColumns.ExpiresAt), boil.Infer())

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
			return nil
		},
	},
	{
		Command:     "coinflip",
		Category:    command.CategoryEconomy,
		Aliases:     []string{"cf", "flip"},
		Description: "Flips a coin. Head or tails. Payout is equal to `<Bet>`",
		Args: []*dispatch.Arg{
			{Name: "Bet", Type: &command.BetArg{Min: 1}},
			{Name: "Coin", Type: &dispatch.StringArg{Options: []string{"Heads", "Tails"}}},
		},
		ArgumentCombos: [][]int{
			{0, 1},
			{1, 0},
		},
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)
			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Author.Username, IconURL: data.Author.AvatarURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			cooldownActive, cooldown := commandCooldown(guildConfig, data.Author.ID, "coinflip")
			if cooldownActive {
				return fmt.Errorf("This command is on cooldown for <t:%d:R>", cooldown)
			}

			fullEconomyMember := getFullEconomyMember(guildConfig, data.Author.ID)
			cash := fullEconomyMember.EconomyUser.Cash

			bet := betAmount(guildConfig, fullEconomyMember, data.ParsedArgs[0].Value)

			if bet > cash {
				return fmt.Errorf("You can't bet more than you have in your hand. You currently have %s%s", guildConfig.EconomySymbol, humanize.Comma(fullEconomyMember.EconomyUser.Cash))
			}

			if guildConfig.EconomyMaxBet > 0 && bet > guildConfig.EconomyMaxBet {
				return fmt.Errorf("You can't bet more than the servers limit. The limit is %s%s", guildConfig.EconomySymbol, humanize.Comma(guildConfig.EconomyMaxBet))
			}

			coinSide := data.ParsedArgs[1].Value.(string)
			if rand.Int63n(2) == 1 {
				cash = cash + bet
				embed.Description = fmt.Sprintf("You flipped %s and won %s%s", coinSide, guildConfig.EconomySymbol, humanize.Comma(bet))
				embed.Color = common.SuccessGreen
			} else {
				cash = cash + bet
				embed.Description = fmt.Sprintf("You flipped %s and lost", coinSide)
			}

			userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: data.Author.ID, Cash: cash}
			userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash), boil.Infer())
			cooldowns := models.EconomyCooldown{GuildID: data.Guild.ID, UserID: data.Author.ID, Type: "coinflip", ExpiresAt: null.Time{Time: time.Now().Add(300 * time.Second), Valid: true}}
			cooldowns.Upsert(context.Background(), common.PQ, true, []string{models.EconomyCooldownColumns.GuildID, models.EconomyCooldownColumns.UserID, models.EconomyCooldownColumns.Type}, boil.Whitelist(models.EconomyCooldownColumns.ExpiresAt), boil.Infer())

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
			return nil
		},
	},
	{
		Command:      "rollnumber",
		Category:     command.CategoryEconomy,
		Aliases:      []string{"roll", "rollnum"},
		Description:  "Rolls a number\n**100** = payout of `<bet>*5`\n**90-99** = payout of `<Bet>*3`\n**65-89** = payout of `<Bet>`\n**64 and under** = Loss of `<Bet>`",
		ArgsRequired: 1,
		Args: []*dispatch.Arg{
			{Name: "Bet", Type: &command.BetArg{Min: 1}},
		},
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)
			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Author.Username, IconURL: data.Author.AvatarURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			cooldownActive, cooldown := commandCooldown(guildConfig, data.Author.ID, "rollnumber")
			if cooldownActive {
				return fmt.Errorf("This command is on cooldown for <t:%d:R>", cooldown)
			}

			fullEconomyMember := getFullEconomyMember(guildConfig, data.Author.ID)
			cash := fullEconomyMember.EconomyUser.Cash

			bet := betAmount(guildConfig, fullEconomyMember, data.ParsedArgs[0].Value)

			if bet > fullEconomyMember.EconomyUser.Cash {
				return fmt.Errorf("You can't bet more than you have in your hand. You currently have %s%s", guildConfig.EconomySymbol, humanize.Comma(fullEconomyMember.EconomyUser.Cash))
			}

			if guildConfig.EconomyMaxBet > 0 && bet > guildConfig.EconomyMaxBet {
				return fmt.Errorf("You can't bet more than the servers limit. The limit is %s%s", guildConfig.EconomySymbol, humanize.Comma(guildConfig.EconomyMaxBet))
			}

			roll := rand.Int63n(100) + 1
			var multiplier int64 = 0
			embed.Color = common.SuccessGreen
			condition := "won"

			switch {
			case roll == 100:
				multiplier = 5
			case roll < 100:
				multiplier = 3
			case roll < 90:
				multiplier = 1
			case roll < 65:
				multiplier = -1
				condition = "lost"
				embed.Color = common.ErrorRed
			}
			cash += bet * multiplier

			embed.Description = fmt.Sprintf("The ball landed on %d, and you %s %s%s", roll, condition, guildConfig.EconomySymbol, humanize.Comma(bet))
			userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: data.Author.ID, Cash: cash}
			userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash), boil.Infer())
			cooldowns := models.EconomyCooldown{GuildID: data.Guild.ID, UserID: data.Author.ID, Type: "rollnumber", ExpiresAt: null.Time{Time: time.Now().Add(300 * time.Second), Valid: true}}
			cooldowns.Upsert(context.Background(), common.PQ, true, []string{models.EconomyCooldownColumns.GuildID, models.EconomyCooldownColumns.UserID, models.EconomyCooldownColumns.Type}, boil.Whitelist(models.EconomyCooldownColumns.ExpiresAt), boil.Infer())

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
			return nil
		},
	},
	{
		Command:     "russianroulette",
		Category:    command.CategoryEconomy,
		Aliases:     []string{"rr"},
		Description: "Russian roulette with up to 6 people\nAll players must join with the same bet\nPayout for winners is `(<Bet>*Players)/winners`",
		Args: []*dispatch.Arg{
			{Name: "Bet", Type: &command.BetArg{Min: 1}},
		},
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)
			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Author.Username, IconURL: data.Author.AvatarURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			fullEconomyMember := getFullEconomyMember(guildConfig, data.Author.ID)
			cash := fullEconomyMember.EconomyUser.Cash

			bet := betAmount(guildConfig, fullEconomyMember, data.ParsedArgs[0])

			if bet > fullEconomyMember.EconomyUser.Cash {
				return fmt.Errorf("You can't bet more than you have in your hand. You currently have %s%s", guildConfig.EconomySymbol, humanize.Comma(fullEconomyMember.EconomyUser.Cash))
			}

			if guildConfig.EconomyMaxBet > 0 && bet > guildConfig.EconomyMaxBet {
				return fmt.Errorf("You can't bet more than the servers limit. The limit is %s%s", guildConfig.EconomySymbol, humanize.Comma(guildConfig.EconomyMaxBet))
			}

			game, exists := activeGames[data.Guild.ID]
			if !exists {
				cooldownActive, cooldown := commandCooldown(guildConfig, data.Author.ID, "russianroulette")
				if cooldownActive {
					return fmt.Errorf("This command is on cooldown for <t:%d:R>", cooldown)
				}

				activeGames[data.Guild.ID] = &RouletteGame{
					HostID:    data.Author.ID,
					Bet:       bet,
					PlayerIDs: []string{data.Author.ID},
					IsActive:  false,
				}

				embed.Description = fmt.Sprintf("A new game of russian roulette has started\n\nTo join, use the command `russianroulette %d` (1/6)\nThis game will automatically start in <t:%d:R> minutes if enough players join.", bet, (time.Now().Unix() + int64((2 * time.Minute).Seconds())))
				embed.Color = common.SuccessGreen
				cash = cash - bet

				userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: data.Author.ID, Cash: cash}
				userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash), boil.Infer())
				cooldowns := models.EconomyCooldown{GuildID: data.Guild.ID, UserID: data.Author.ID, Type: "russianroulette", ExpiresAt: null.Time{Time: time.Now().Add(300 * time.Second), Valid: true}}
				cooldowns.Upsert(context.Background(), common.PQ, true, []string{models.EconomyCooldownColumns.GuildID, models.EconomyCooldownColumns.UserID, models.EconomyCooldownColumns.Type}, boil.Whitelist(models.EconomyCooldownColumns.ExpiresAt), boil.Infer())

				functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
				time.AfterFunc(2*time.Minute, func() { startGame(guildConfig, data.Guild.ID, data.Channel.ID) })
				return nil
			}

			if bet != game.Bet {
				return fmt.Errorf("All players must bet the same amount. This game's bet is %s%s", guildConfig.EconomySymbol, humanize.Comma(game.Bet))
			}

			if game.IsActive {
				return errors.New("You can't join or start a game while there is one running")
			}

			if slices.Contains(game.PlayerIDs, data.Author.ID) {
				return errors.New("You've already joined this game.\nIf you're the host, start the game with `russianroulette start`")
			}

			game.PlayerIDs = append(game.PlayerIDs, data.Author.ID)
			if len(game.PlayerIDs) == 6 {
				startGame(guildConfig, data.Guild.ID, data.Channel.ID)
				return nil
			}

			embed.Description = fmt.Sprintf("You've joined this game of russian roulette with a bet of %s%s (%d/6)", guildConfig.EconomySymbol, humanize.Comma(bet), len(game.PlayerIDs))
			embed.Color = common.SuccessGreen
			cash = cash - bet

			userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: data.Author.ID, Cash: cash}
			userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash), boil.Infer())

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
			return nil
		},
	},
	{
		Command:      "snakeeyes",
		Category:     command.CategoryEconomy,
		Aliases:      []string{"dice"},
		Description:  "Rolls 2 6-sided dice, with a payout of `<Bet>*36` if they both land on 1",
		ArgsRequired: 1,
		Args: []*dispatch.Arg{
			{Name: "Bet", Type: &command.BetArg{Min: 1}},
		},
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)
			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Author.Username, IconURL: data.Author.AvatarURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			cooldownActive, cooldown := commandCooldown(guildConfig, data.Author.ID, "snakeeyes")
			if cooldownActive {
				return fmt.Errorf("This command is on cooldown for <t:%d:R>", cooldown)
			}

			fullEconomyMember := getFullEconomyMember(guildConfig, data.Author.ID)
			cash := fullEconomyMember.EconomyUser.Cash

			bet := betAmount(guildConfig, fullEconomyMember, data.ParsedArgs[0])

			if bet > cash {
				return fmt.Errorf("You can't bet more than you have in your hand. You currently have %s%s", guildConfig.EconomySymbol, humanize.Comma(fullEconomyMember.EconomyUser.Cash))
			}
			if guildConfig.EconomyMaxBet > 0 && bet > guildConfig.EconomyMaxBet {
				return fmt.Errorf("You can't bet more than the servers limit. The limit is %s%s", guildConfig.EconomySymbol, humanize.Comma(guildConfig.EconomyMaxBet))
			}

			d1, d2 := rand.Int63n(6)+1, rand.Int63n(6)+1
			condition := "won"
			if d1 == 1 && d2 == 1 {
				cash = cash + (bet * 36)
			} else {
				cash = cash - bet
				condition = "lost"
			}
			embed.Description = fmt.Sprintf("You rolled %d & %d, and you %s %s%s", d1, d2, condition, guildConfig.EconomySymbol, humanize.Comma(bet))

			userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: data.Author.ID, Cash: cash}
			userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash), boil.Infer())
			cooldowns := models.EconomyCooldown{GuildID: data.Guild.ID, UserID: data.Author.ID, Type: "snakeeyes", ExpiresAt: null.Time{Time: time.Now().Add(300 * time.Second), Valid: true}}
			cooldowns.Upsert(context.Background(), common.PQ, true, []string{models.EconomyCooldownColumns.GuildID, models.EconomyCooldownColumns.UserID, models.EconomyCooldownColumns.Type}, boil.Whitelist(models.EconomyCooldownColumns.ExpiresAt), boil.Infer())

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
			return nil
		},
	},
}

var transferCommands = []*dispatch.Command{
	{
		Command:      "deposit",
		Category:     command.CategoryEconomy,
		Aliases:      []string{"dep"},
		Description:  "Deposits a given amount into your bank",
		ArgsRequired: 1,
		Args: []*dispatch.Arg{
			{Name: "Amount", Type: &command.BetArg{Min: 1}},
		},
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)
			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Author.Username, IconURL: data.Author.AvatarURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			fullEconomyMember := getFullEconomyMember(guildConfig, data.Author.ID)
			cash := fullEconomyMember.EconomyUser.Cash
			bank := fullEconomyMember.EconomyUser.Bank

			var depositAmount int64
			switch data.ParsedArgs[0].Value.(type) {
			case string:
				depositAmount = fullEconomyMember.EconomyUser.Cash
			default:
				depositAmount = data.ParsedArgs[0].Value.(int64)
			}

			if depositAmount > fullEconomyMember.EconomyUser.Cash {
				return fmt.Errorf("You're unable to deposit more than you have in cash\nYou currently have %s%s", guildConfig.EconomySymbol, humanize.Comma(cash))
			}

			cash = cash - depositAmount
			bank = bank + depositAmount

			embed.Description = fmt.Sprintf("You deposited %s%s into your bank\nThere is now %s%s in your bank", guildConfig.EconomySymbol, humanize.Comma(depositAmount), guildConfig.EconomySymbol, humanize.Comma(bank))
			embed.Color = common.SuccessGreen

			userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: data.Author.ID, Cash: cash, Bank: bank}
			userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash, models.EconomyUserColumns.Bank), boil.Infer())

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
			return nil
		},
	},
	{
		Command:     "withdraw",
		Category:    command.CategoryEconomy,
		Aliases:     []string{"with"},
		Description: "Withdraws a given amount from your bank",
		Args: []*dispatch.Arg{
			{Name: "Amount", Type: &command.BetArg{Min: 1}},
		},
		ArgsRequired: 1,
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)
			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Author.Username, IconURL: data.Author.AvatarURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			fullEconomyMember := getFullEconomyMember(guildConfig, data.Author.ID)
			cash := fullEconomyMember.EconomyUser.Cash
			bank := fullEconomyMember.EconomyUser.Bank

			var withdrawAmount int64
			switch data.ParsedArgs[0].Value.(type) {
			case string:
				withdrawAmount = fullEconomyMember.EconomyUser.Bank
			default:
				withdrawAmount = data.ParsedArgs[0].Value.(int64)
			}

			if withdrawAmount > bank {
				return fmt.Errorf("You're unable to withdraw more than you have in your bank\nYou currently have %s%s", guildConfig.EconomySymbol, humanize.Comma(bank))
			}
			if bank < 0 {
				return fmt.Errorf("You're unable to withdraw from your overdraft\nYou are currently %s%s in arrears", guildConfig.EconomySymbol, humanize.Comma(bank))
			}

			cash = cash + withdrawAmount
			bank = bank - withdrawAmount

			embed.Description = fmt.Sprintf("You've withdrawn %s%s from your bank\nThere is now %s%s in your bank", guildConfig.EconomySymbol, humanize.Comma(withdrawAmount), guildConfig.EconomySymbol, humanize.Comma(bank))
			embed.Color = common.SuccessGreen

			userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: data.Author.ID, Cash: cash, Bank: bank}
			userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash, models.EconomyUserColumns.Bank), boil.Infer())

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
			return nil
		},
	},
	{
		Command:      "givemoney",
		Category:     command.CategoryEconomy,
		Aliases:      []string{"loan"},
		Description:  "Gives money to a specified member from your cash",
		ArgsRequired: 2,
		Args: []*dispatch.Arg{
			{Name: "Member", Type: dispatch.Member},
			{Name: "Amount", Type: &command.BetArg{Min: 1}},
		},
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)
			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Author.Username, IconURL: data.Author.AvatarURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			fullEconomyMember := getFullEconomyMember(guildConfig, data.Author.ID)
			cash := fullEconomyMember.EconomyUser.Cash

			target := data.ParsedArgs[0].Value.(*discordgo.Member)

			var giveAmount int64
			switch data.ParsedArgs[1].Value.(type) {
			case string:
				giveAmount = fullEconomyMember.EconomyUser.Cash
			default:
				giveAmount = data.ParsedArgs[1].Value.(int64)
			}

			if giveAmount > fullEconomyMember.EconomyUser.Cash {
				return fmt.Errorf("You don't have enough cash to give. You have %s%s", guildConfig.EconomySymbol, humanize.Comma(cash))
			}

			fullEconomyMemberTarget := getFullEconomyMember(guildConfig, target.User.ID)
			targetCash := fullEconomyMemberTarget.EconomyUser.Cash

			cash = cash - giveAmount
			targetCash = targetCash + giveAmount

			embed.Description = fmt.Sprintf("You gave %s%s to %s!", guildConfig.EconomySymbol, humanize.Comma(functions.ToInt64(giveAmount)), target.Mention())
			embed.Color = common.SuccessGreen

			userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: data.Author.ID, Cash: cash}
			userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash), boil.Infer())
			receivingEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: target.User.ID, Cash: targetCash}
			receivingEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash), boil.Infer())

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
			return nil
		},
	},
	{
		Command:           "addmoney",
		Category:          command.CategoryEconomy,
		Description:       "Adds money to a specified users cash/bank balance",
		RequiredUserPerms: []int64{discordgo.PermissionManageGuild},
		Args: []*dispatch.Arg{
			{Name: "Member", Type: dispatch.Member},
			{Name: "Amount", Type: &command.BetArg{Min: 1}},
			{Name: "Place", Type: &dispatch.StringArg{Options: []string{"Cash", "Bank"}}},
		},
		ArgumentCombos: [][]int{
			{0, 1, 2},
			{0, 2, 1},
		},
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)
			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Author.Username, IconURL: data.Author.AvatarURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			member := data.ParsedArgs[0].Value.(*discordgo.Member)
			fullEconomyMember := getFullEconomyMember(guildConfig, member.User.ID)
			cash := fullEconomyMember.EconomyUser.Cash
			bank := fullEconomyMember.EconomyUser.Bank

			var amount int64
			switch data.ParsedArgs[1].Value.(type) {
			case string:
				amount = fullEconomyMember.EconomyUser.Cash
			default:
				amount = data.ParsedArgs[1].Value.(int64)
			}

			if strings.EqualFold(data.ParsedArgs[2].Value.(string), "cash") {
				cash = cash + amount
			} else {
				bank = bank + amount
			}

			embed.Description = fmt.Sprintf("You added %s%s to %ss %s", guildConfig.EconomySymbol, humanize.Comma(amount), member.Mention(), data.ParsedArgs[1].Value.(string))
			embed.Color = common.SuccessGreen

			userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: member.User.ID, Cash: cash, Bank: bank}
			userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash, models.EconomyUserColumns.Bank), boil.Infer())
			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})

			return nil
		},
	},
	{
		Command:           "removemoney",
		Category:          command.CategoryEconomy,
		Description:       "Removes money from a specified users cash/bank balance",
		RequiredUserPerms: []int64{discordgo.PermissionManageGuild},
		Args: []*dispatch.Arg{
			{Name: "Member", Type: dispatch.Member},
			{Name: "Amount", Type: &command.BetArg{Min: 1}},
			{Name: "Place", Type: &dispatch.StringArg{Options: []string{"Cash", "Bank"}}},
		},
		ArgumentCombos: [][]int{
			{0, 1, 2},
			{0, 2, 1},
		},
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)
			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Author.Username, IconURL: data.Author.AvatarURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			member := data.ParsedArgs[0].Value.(*discordgo.Member)
			fullEconomyMember := getFullEconomyMember(guildConfig, member.User.ID)
			cash := fullEconomyMember.EconomyUser.Cash
			bank := fullEconomyMember.EconomyUser.Bank

			var amount int64
			switch data.ParsedArgs[1].Value.(type) {
			case string:
				amount = fullEconomyMember.EconomyUser.Cash
			default:
				amount = data.ParsedArgs[1].Value.(int64)
			}

			if strings.EqualFold(data.ParsedArgs[2].Value.(string), "cash") {
				cash = cash + amount
			} else {
				bank = bank + amount
			}

			embed.Description = fmt.Sprintf("You removed %s%s from %ss %s", guildConfig.EconomySymbol, humanize.Comma(amount), member.Mention(), data.ParsedArgs[1].Value.(string))
			embed.Color = common.SuccessGreen

			userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: member.User.ID, Cash: cash, Bank: bank}
			userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash, models.EconomyUserColumns.Bank), boil.Infer())

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
			return nil
		},
	},
}

func startGame(config *Config, guildID, channelID string) {
	game, exists := activeGames[guildID]
	if !exists {
		return
	}

	game.IsActive = true
	hostMember, _ := functions.GetUser(game.HostID)

	embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: hostMember.Username, IconURL: hostMember.AvatarURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

	removeLeftPlayers(guildID, game)
	if len(game.PlayerIDs) <= 1 {
		functions.SendBasicMessage(channelID, "There weren't enough players to start russian roulette. Please start a new one.")
		delete(activeGames, guildID)
		return
	}

	functions.SendBasicMessage(channelID, "The russian roulette game has begun")
	loser, _ := functions.GetMember(config.GuildID, game.PlayerIDs[rand.Intn(len(game.PlayerIDs))])
	for _, player := range game.PlayerIDs {
		member, _ := functions.GetMember(guildID, player)
		if player != loser.User.ID {
			functions.SendBasicMessage(channelID, fmt.Sprintf("> **%s** pulled the trigger and survived", member.Mention()))
			continue
		}
		functions.SendBasicMessage(channelID, fmt.Sprintf("> **%s** pulled the trigger and died", member.Mention()))
		break
	}

	fields := []*discordgo.MessageEmbedField{}
	payout := (game.Bet * int64(len(game.PlayerIDs))) / int64(len(game.PlayerIDs)-1)
	for _, playerID := range game.PlayerIDs {
		if loser.User.ID == playerID {
			continue
		}

		fullEconomyMember := getFullEconomyMember(config, playerID)
		cash := fullEconomyMember.EconomyUser.Cash

		member, _ := functions.GetMember(guildID, playerID)
		field := &discordgo.MessageEmbedField{Name: member.User.Username, Value: member.Mention(), Inline: false}
		fields = append(fields, field)

		userEntry := models.EconomyUser{GuildID: guildID, UserID: playerID, Cash: cash + payout}
		userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash), boil.Infer())
	}

	embed.Title = "Russian roulette winners"
	embed.Description = fmt.Sprintf("Payout is %s%s per winner", config.EconomySymbol, humanize.Comma(payout))
	embed.Fields = fields
	embed.Color = common.SuccessGreen

	delete(activeGames, guildID)
	functions.SendMessage(channelID, &discordgo.MessageSend{Embed: embed})
}

func removeLeftPlayers(guildID string, game *RouletteGame) {
	filtered := make([]string, 0, len(game.PlayerIDs))
	for _, playerID := range game.PlayerIDs {
		if _, err := functions.GetMember(guildID, playerID); err == nil {
			filtered = append(filtered, playerID)
		}
	}
	game.PlayerIDs = filtered
}

var shopCommands = []*dispatch.Command{
	{
		Command:     "shop",
		Category:    command.CategoryEconomy,
		Description: "Views the shop for the server",
		Args: []*dispatch.Arg{
			{Name: "Page", Type: &dispatch.IntArg{Min: 1}},
		},
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)

			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Guild.Name + " Shop", IconURL: data.Guild.IconURL("256")}, Description: "No items are in the shop for this page.\nAdd some with `createitem`", Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}
			components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "previous", Style: 4, Disabled: true, CustomID: "shop_back"}, discordgo.Button{Label: "next", Style: 3, Disabled: true, CustomID: "shop_forward"}}}}

			page := 1
			if len(data.ParsedArgs) > 0 {
				page = getPageNumber(data.ParsedArgs[0].Value.(int))
			}

			offset := (page - 1) * 10
			guildShop, err := models.EconomyShops(models.EconomyShopWhere.GuildID.EQ(data.Guild.ID), qm.OrderBy("price DESC"), qm.Offset(offset)).All(context.Background(), common.PQ)
			if err != nil || len(guildShop) == 0 {
				sendPaginatedEmbed(data.Channel.ID, embed, components, guildShop, page)
			}

			embed.Description = "Buy an item with `buyitem <Name> [Quantity:Int]`\nFor more information about an item, use `iteminfo <Name>`"
			embed.Color = common.SuccessGreen
			fields := []*discordgo.MessageEmbedField{}
			for i, item := range guildShop {
				if i == 10 {
					break
				}

				quantity := "Infinite"
				if item.Quantity > 0 {
					quantity = humanize.Comma(item.Quantity)
				}

				price := humanize.Comma(item.Price)

				fieldName := fmt.Sprintf("%s%s - %s - %s", guildConfig.EconomySymbol, price, item.Name, quantity)
				fieldDesc := item.Description

				itemField := &discordgo.MessageEmbedField{Name: fieldName, Value: fieldDesc, Inline: false}
				fields = append(fields, itemField)
			}
			embed.Fields = fields

			sendPaginatedEmbed(data.Channel.ID, embed, components, guildShop, page)
			return nil
		},
	},
	{
		Command:      "iteminfo",
		Category:     command.CategoryEconomy,
		Description:  "Views the saved information about an item",
		ArgsRequired: 1,
		Args: []*dispatch.Arg{
			{Name: "Name", Type: dispatch.String},
		},
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)

			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Guild.Name + " Store", IconURL: data.Guild.IconURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			item, err := models.EconomyShops(models.EconomyShopWhere.GuildID.EQ(data.Guild.ID), models.EconomyShopWhere.Name.EQ(data.ParsedArgs[0].Value.(string))).One(context.Background(), common.PQ)
			if err != nil {
				return errors.New("This item does not exist")
			}

			embed.Color = common.SuccessGreen
			embed.Fields = []*discordgo.MessageEmbedField{
				{Name: "Name", Value: item.Name, Inline: true},
				{Name: "⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀", Value: "⠀⠀", Inline: true},
				{Name: "Price", Value: fmt.Sprintf("%s%d", guildConfig.EconomySymbol, item.Price), Inline: true},
				{Name: "Description", Value: item.Description, Inline: true},
				{Name: "⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀", Value: "⠀⠀", Inline: true},
				{Name: "Quantity", Value: humanize.Comma(item.Quantity), Inline: true},
				{Name: "Role given", Value: fmt.Sprintf("<@&%s>", item.Role), Inline: false},
				{Name: "Reply message", Value: item.Reply, Inline: true},
			}

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})

			return nil
		},
	},
	{
		Command:      "buyitem",
		Category:     command.CategoryEconomy,
		Aliases:      []string{"buy"},
		Description:  "Buys an item from the shop",
		ArgsRequired: 1,
		Args: []*dispatch.Arg{
			{Name: "Name", Type: dispatch.String},
			{Name: "Quantity", Type: &command.BetArg{Min: 1}},
		},
		Run: func(data *dispatch.Data) error {
			guildConfig := GetConfig(data.Guild.ID)

			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Guild.Name + " Store", IconURL: data.Guild.IconURL("256")}, Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}

			fullEconomyMember := getFullEconomyMember(guildConfig, data.Author.ID)
			cash := fullEconomyMember.EconomyUser.Cash

			item, err := models.EconomyShops(models.EconomyShopWhere.GuildID.EQ(data.Guild.ID), models.EconomyShopWhere.Name.EQ(data.ParsedArgs[0].Value.(string))).One(context.Background(), common.PQ)
			if err != nil {
				return errors.New("This item doesn't exist. Use `shop` to view all items")
			}

			var buyQuantity int64 = 1
			if len(data.ParsedArgs) > 1 {
				switch data.ParsedArgs[1].Value.(type) {
				case string:
					quantity := map[string]int64{"max": (cash / item.Price), "all": item.Quantity}
					buyQuantity = quantity[data.ParsedArgs[1].Value.(string)]
					if buyQuantity == 0 {
						buyQuantity = quantity["max"]
					}
				default:
					buyQuantity = data.ParsedArgs[1].Value.(int64)
				}

				if item.Quantity > 0 && buyQuantity > item.Quantity {
					return errors.New("There's not enough of this in the shop to buy that much")
				}
			}

			if (item.Name == "Chicken") && buyQuantity > 1 {
				return errors.New("You can't have more than one chicken at a time")
			}

			for _, inventoryItem := range fullEconomyMember.EconomyUserInventory {
				if inventoryItem.Name != "Chicken" {
					continue
				}

				return errors.New("You can't have more than one chicken at a time")
			}

			if (item.Price * buyQuantity) > cash {
				return errors.New("You don't have enough money to buy this item")
			}

			newQuantity := item.Quantity - buyQuantity
			if item.Quantity > 0 && newQuantity == 0 {
				item.Delete(context.Background(), common.PQ)
			} else if item.Quantity != 0 {
				item.Quantity = newQuantity
				item.Update(context.Background(), common.PQ, boil.Infer())
			}

			newQuantity = buyQuantity
			for _, inventoryItem := range fullEconomyMember.EconomyUserInventory {
				if inventoryItem.Name != item.Name {
					continue
				}

				newQuantity = inventoryItem.Quantity + buyQuantity
				break
			}

			embed.Color = common.SuccessGreen
			embed.Description = fmt.Sprintf("You bought %s of %s for %s%s", humanize.Comma(buyQuantity), item.Name, guildConfig.EconomySymbol, humanize.Comma(item.Price*buyQuantity))
			cash = cash - (item.Price * buyQuantity)

			userInventory := models.EconomyUserInventory{GuildID: data.Guild.ID, UserID: data.Author.ID, Name: item.Name, Description: item.Description, Quantity: newQuantity, Role: item.Role, Reply: item.Reply}
			userInventory.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserInventoryColumns.GuildID, models.EconomyUserInventoryColumns.UserID, models.EconomyUserInventoryColumns.Name}, boil.Whitelist(models.EconomyUserInventoryColumns.Quantity), boil.Infer())
			userEntry := models.EconomyUser{GuildID: data.Guild.ID, UserID: data.Author.ID, Cash: cash}
			userEntry.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserColumns.GuildID, models.EconomyUserColumns.UserID}, boil.Whitelist(models.EconomyUserColumns.Cash), boil.Infer())

			functions.SendMessage(data.Channel.ID, &discordgo.MessageSend{Embed: embed})
			return nil
		},
	},
}

func shopPagination(s *discordgo.Session, b *discordgo.InteractionCreate) {
	if b.MessageComponentData().CustomID != "shop_back" && b.MessageComponentData().CustomID != "shop_forward" {
		return
	}

	guildConfig := GetConfig(b.GuildID)
	guild, err := common.Session.Guild(b.GuildID)
	if err != nil || guild == nil {
		return
	}
	embed := []*discordgo.MessageEmbed{{Author: &discordgo.MessageEmbedAuthor{Name: guild.Name + " Shop", IconURL: guild.IconURL("256")}, Description: "No items are in the shop for this page.\nAdd some with `createitem`", Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}}
	components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "previous", Style: 4, Disabled: true, CustomID: "shop_back"}, discordgo.Button{Label: "next", Style: 3, Disabled: true, CustomID: "shop_forward"}}}}

	re := regexp.MustCompile(`\d+`)
	page, _ := strconv.Atoi(re.FindString(b.Message.Embeds[0].Footer.Text))
	switch b.MessageComponentData().CustomID {
	case "shop_forward":
		page++
	case "shop_back":
		page--
	}

	offset := (page - 1) * 10
	guildShop, err := models.EconomyShops(models.EconomyShopWhere.GuildID.EQ(b.GuildID), qm.OrderBy("price DESC"), qm.Offset(offset)).All(context.Background(), common.PQ)
	if err == nil && len(guildShop) != 0 {
		embed[0].Description = "Buy an item with `buyitem <Name> [Quantity:Int]`\nFor more information about an item, use `iteminfo <Name>`"
		embed[0].Color = common.SuccessGreen
	}

	fields := []*discordgo.MessageEmbedField{}
	for i, item := range guildShop {
		if i == 10 {
			break
		}

		quantity := "Infinite"
		if item.Quantity > 0 {
			quantity = humanize.Comma(item.Quantity)
		}
		price := humanize.Comma(item.Price)

		fieldName := fmt.Sprintf("%s%s - %s - %s", guildConfig.EconomySymbol, price, item.Name, quantity)
		fieldDesc := item.Description

		itemField := &discordgo.MessageEmbedField{Name: fieldName, Value: fieldDesc, Inline: false}
		fields = append(fields, itemField)
	}

	embed[0].Fields = fields
	embed[0].Footer = &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Page: %d", page)}
	if page != 1 {
		row := components[0].(discordgo.ActionsRow)
		btnPrev := row.Components[0].(discordgo.Button)
		btnPrev.Disabled = false
		row.Components[0] = btnPrev
		components[0] = row
	}
	if len(guildShop) > offset {
		row := components[0].(discordgo.ActionsRow)
		btnNext := row.Components[1].(discordgo.Button)
		btnNext.Disabled = false
		row.Components[1] = btnNext
		components[0] = row
	}

	common.Session.InteractionRespond(b.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Embeds: embed, Components: components}})
}

var inventoryCommands = []*dispatch.Command{
	{
		Command:     "inventory",
		Category:    command.CategoryEconomy,
		Aliases:     []string{"inv"},
		Description: "Your inventory",
		Args: []*dispatch.Arg{
			{Name: "Page", Type: &dispatch.IntArg{Min: 1}},
		},
		Run: func(data *dispatch.Data) error {
			embed := &discordgo.MessageEmbed{Author: &discordgo.MessageEmbedAuthor{Name: data.Author.Username + " Inventory", IconURL: data.Author.AvatarURL("256")}, Description: "There are no item on this page\nBuy some with `buyitem`", Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}
			components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "previous", Style: 4, Disabled: true, CustomID: "inventory_back"}, discordgo.Button{Label: "next", Style: 3, Disabled: true, CustomID: "inventory_forward"}}}}

			page := 1
			if len(data.ParsedArgs) > 0 {
				page = getPageNumber(data.ParsedArgs[0].Value.(int))
			}

			offset := (page - 1) * 10
			userInventory, err := models.EconomyUserInventories(models.EconomyUserInventoryWhere.GuildID.EQ(data.Guild.ID), models.EconomyUserInventoryWhere.UserID.EQ(data.Author.ID), qm.OrderBy("quantity DESC"), qm.Offset(offset)).All(context.Background(), common.PQ)
			if err != nil || len(userInventory) == 0 {
				sendPaginatedEmbed(data.Channel.ID, embed, components, userInventory, page)
			}

			embed.Description = "Use an item with `useitem <Name>`\nFor more information about an item, use `iteminfo <Name>`"
			embed.Color = common.SuccessGreen
			fields := []*discordgo.MessageEmbedField{}
			for i, item := range userInventory {
				if i == 10 {
					break
				}

				role := "None"
				if item.Role != "0" {
					role = "<@&" + item.Role + ">"
				}

				itemField := &discordgo.MessageEmbedField{Name: item.Name, Value: fmt.Sprintf("Description: %s\nQuantity: %s\nRole given: %s", item.Description, humanize.Comma(item.Quantity), role), Inline: false}
				fields = append(fields, itemField)
			}

			embed.Fields = fields
			embed.Footer = &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Page: %d", page)}

			sendPaginatedEmbed(data.Channel.ID, embed, components, userInventory, page)

			return nil
		},
	},
	{
		Command:      "useitem",
		Category:     command.CategoryEconomy,
		Aliases:      []string{"use"},
		Description:  "Uses an item present in your inventory",
		ArgsRequired: 1,
		Args: []*dispatch.Arg{
			{Name: "Name", Type: dispatch.String},
			{Name: "Quantity", Type: &dispatch.IntArg{Min: 1}},
		},
		Run: func(data *dispatch.Data) error {
			guild, err := common.Session.Guild(data.Guild.ID)
			if err != nil || guild == nil {
				return errors.New("Failed to retrieve guild information.")
			}

			item, err := models.EconomyUserInventories(models.EconomyUserInventoryWhere.GuildID.EQ(data.Guild.ID), models.EconomyUserInventoryWhere.UserID.EQ(data.Author.ID), models.EconomyUserInventoryWhere.Name.EQ(data.ParsedArgs[0].Value.(string))).One(context.Background(), common.PQ)
			if err != nil {
				return errors.New("You don't have this item\nUse `inventory [Page]` to view your items")
			}

			if _, err := functions.GetRole(data.Guild.ID, item.Role); err == nil {
				functions.AddRole(data.Guild.ID, data.Author.ID, item.Role)
			}

			quantity := item.Quantity
			newQuantity := quantity - 1
			if newQuantity == 0 {
				item.Delete(context.Background(), common.PQ)
			} else if newQuantity > 0 {
				item.Quantity = newQuantity
				item.Upsert(context.Background(), common.PQ, true, []string{models.EconomyUserInventoryColumns.GuildID, models.EconomyUserInventoryColumns.UserID, models.EconomyUserInventoryColumns.Name}, boil.Whitelist(models.EconomyUserInventoryColumns.Quantity), boil.Infer())
			}

			functions.SendBasicMessage(data.Channel.ID, item.Reply, 10)
			return nil
		},
	},
}

func inventoryPagination(s *discordgo.Session, b *discordgo.InteractionCreate) {
	if b.MessageComponentData().CustomID != "inventory_back" && b.MessageComponentData().CustomID != "inventory_forward" {
		return
	}

	embed := []*discordgo.MessageEmbed{{Author: &discordgo.MessageEmbedAuthor{Name: b.Member.User.Username + " inventory", IconURL: b.Member.User.AvatarURL("256")}, Description: "There are no item on this page\nBuy some with `buyitem`", Timestamp: time.Now().Format(time.RFC3339), Color: common.ErrorRed}}
	components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "previous", Style: 4, Disabled: true, CustomID: "inventory_back"}, discordgo.Button{Label: "next", Style: 3, Disabled: true, CustomID: "inventory_forward"}}}}

	re := regexp.MustCompile(`\d+`)
	page, _ := strconv.Atoi(re.FindString(b.Message.Embeds[0].Footer.Text))
	switch b.MessageComponentData().CustomID {
	case "inventory_forward":
		page++
	case "inventory_back":
		page--
	}

	offset := (page - 1) * 10
	userInventory, err := models.EconomyUserInventories(models.EconomyUserInventoryWhere.GuildID.EQ(b.GuildID), models.EconomyUserInventoryWhere.UserID.EQ(b.Member.User.ID), qm.OrderBy("quantity DESC"), qm.Offset(offset)).All(context.Background(), common.PQ)
	if err == nil && len(userInventory) != 0 {
		embed[0].Description = "Use an item with `useitem <Name>`\nFor more information about an item, use `iteminfo <Name>`"
		embed[0].Color = common.SuccessGreen
	}

	fields := []*discordgo.MessageEmbedField{}
	for i, item := range userInventory {
		if i == 10 {
			break
		}

		role := "None"
		if item.Role != "0" {
			role = "<@&" + item.Role + ">"
		}

		itemField := &discordgo.MessageEmbedField{Name: item.Name, Value: fmt.Sprintf("Description: %s\nQuantity: %s\nRole given: %s", item.Description, humanize.Comma(item.Quantity), role), Inline: false}
		fields = append(fields, itemField)
	}

	embed[0].Fields = fields
	embed[0].Footer = &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Page: %d", page)}
	if page != 1 {
		row := components[0].(discordgo.ActionsRow)
		btnPrev := row.Components[0].(discordgo.Button)
		btnPrev.Disabled = false
		row.Components[0] = btnPrev
		components[0] = row
	}
	if len(userInventory) > offset {
		row := components[0].(discordgo.ActionsRow)
		btnNext := row.Components[1].(discordgo.Button)
		btnNext.Disabled = false
		row.Components[1] = btnNext
		components[0] = row
	}

	common.Session.InteractionRespond(b.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Embeds: embed, Components: components}})
}
