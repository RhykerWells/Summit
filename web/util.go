package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/common"
	"github.com/bwmarrin/discordgo"
	"golang.org/x/oauth2"
)

var (
	ErrFailedRetrievingInfo = errors.New("failed_retrieving_info")
	ErrDiscordAPIError      = errors.New("discord_api_error")
	ErrJSONDecodeError      = errors.New("json_decode_error")
)

func SetTmplContextData(ctx context.Context, data TmplContextData) context.Context {
	// Check for existing data
	if val := ctx.Value(CtxKeyTmplData); val != nil {
		cast := val.(TmplContextData)
		for k, v := range data {
			cast[k] = v
		}
		return ctx
	}

	// Fallback
	return context.WithValue(ctx, CtxKeyTmplData, TmplContextData(data))
}

// tokenToUser converts an OAuth token to a valid discordgo.User object
func tokenToUser(ctx context.Context, token *oauth2.Token) (*discordgo.User, error) {
	client := OauthConf.Client(ctx, token)
	resp, err := client.Get("https://discord.com/api/v10/users/@me")
	if err != nil {
		return nil, ErrFailedRetrievingInfo
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrDiscordAPIError
	}
	defer resp.Body.Close()

	var user *discordgo.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, ErrJSONDecodeError
	}

	return user, nil
}

// isUserManaged returns a boolean of whether or not the user has the permissions to manage the guild
// Permissions required are: Owner, Manage Server or Administrator
func isUserManaged(guildID string, member *discordgo.Member) bool {
	guild, err := common.Session.Guild(guildID)
	if err == nil && guild.OwnerID == member.User.ID {
		return true
	}

	for _, roleID := range member.Roles {
		role, err := common.Session.State.Role(guildID, roleID)
		if err != nil {
			continue
		}

		if (role.Permissions&discordgo.PermissionAdministrator != 0) ||
			(role.Permissions&discordgo.PermissionManageServer != 0) {
			return true
		}
	}

	return false
}

type PartialGuild struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Permissions string `json:"permissions"`
}

// getUserGuilds returns the guild IDs that the user can currently manage
// and the guild IDs that the user has the correct permission to manage if the bot is added
func getUserManagedGuilds(ctx context.Context, token *oauth2.Token) ([]map[string]interface{}, []map[string]interface{}, error) {
	user, err := tokenToUser(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	managedGuilds := make(map[string]string)
	for _, guild := range common.Session.State.Guilds {
		member, err := common.Session.GuildMember(guild.ID, user.ID)
		if err != nil {
			continue
		}

		if managed := isUserManaged(guild.ID, member); managed {
			managedGuilds[guild.ID] = guild.Name
		}
	}

	client := OauthConf.Client(ctx, token)
	resp, err := client.Get("https://discord.com/api/v10/users/@me/guilds")
	if err != nil {
		return nil, nil, ErrFailedRetrievingInfo
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, ErrDiscordAPIError
	}

	defer resp.Body.Close()

	var guilds []PartialGuild

	if err := json.NewDecoder(resp.Body).Decode(&guilds); err != nil {
		return nil, nil, ErrJSONDecodeError
	}

	availableGuilds := make(map[string]string)
	for _, guild := range guilds {
		if _, ok := managedGuilds[guild.ID]; ok {
			continue
		}

		permInt, _ := strconv.Atoi(guild.Permissions)
		var requiredPerms = discordgo.PermissionManageServer | discordgo.PermissionAdministrator
		if permInt&requiredPerms != 0 {
			availableGuilds[guild.ID] = guild.Name
		}
	}

	for id := range managedGuilds {
		delete(availableGuilds, id)
	}

	fullManagedGuilds := getPopulatedGuildList(managedGuilds, URL+"/static/img/icons/question.svg", true)
	fullAvailableGuilds := getPopulatedGuildList(availableGuilds, URL+"/static/img/icons/plus.svg", false)

	return fullManagedGuilds, fullAvailableGuilds, nil
}

// getPopulatedGuildList returns a list of guilds with their ID, name and avatar URL to be used for template data
func getPopulatedGuildList(guilds map[string]string, defaultIcon string, useGuildIcon bool) []map[string]interface{} {
	guildList := make([]map[string]interface{}, 0)
	for guildID, guildName := range guilds {
		avatarURL := defaultIcon
		if useGuildIcon {
			if guild, err := common.Session.Guild(guildID); err == nil {
				if url := guild.IconURL("1024"); url != "" {
					avatarURL = url
				}
			}
		}

		guildList = append(guildList, TmplContextData{
			"ID":     guildID,
			"Avatar": avatarURL,
			"Name":   guildName,
		})
	}

	return guildList
}

func getGuildTmplData(g *discordgo.Guild) TmplContextData {
	guildChannels, _ := common.Session.GuildChannels(g.ID)
	sort.SliceStable(guildChannels, func(i, j int) bool {
		return guildChannels[i].Position < guildChannels[j].Position
	})

	guildRoles, _ := common.Session.GuildRoles(g.ID)
	sort.SliceStable(guildRoles, func(i, j int) bool {
		return guildRoles[i].Position > guildRoles[j].Position
	})

	botMember, _ := common.Session.GuildMember(g.ID, common.Bot.ID)
	botHighestRole := functions.HighestRole(g.ID, botMember)

	base := TmplContextData{
		"ID":                     g.ID,
		"Name":                   g.Name,
		"Avatar":                 g.IconURL("1024"),
		"Channels":               guildChannels,
		"Roles":                  guildRoles,
		"BotHighestRolePosition": botHighestRole.Position,
	}
	if base["Avatar"] == "" {
		base["Avatar"] = URL + "/static/img/icons/question.svg"
	}

	return base
}
