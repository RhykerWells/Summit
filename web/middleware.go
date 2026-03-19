package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/common"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/gorilla/schema"
	"github.com/patrickmn/go-cache"
	"goji.io/v3/pat"
	"golang.org/x/oauth2"
)

var sessionStore = cache.New(24*time.Hour*30, 1*time.Hour)

type CtxKey int

const (
	CtxKeyTmplData CtxKey = iota
	CtxKeyFormParsed
)

// createCSRF generates a CSRF token to be used for validating requests such as logins
func createCSRF() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// setCSRF sets the csrf token in the clients web cache as a cookie
func setCSRF(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:    "summit_csrf",
		Value:   token,
		Path:    "/",
		Expires: time.Now().Add(300 * time.Second),
		Secure:  true,
	})
}

// getCSRF returns the csrf token from the clients cookies
func getCSRF(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("summit_csrf")
	if err == nil {
		return cookie.Value
	}

	// If decoding failed — clear the bad cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "summit_csrf",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
		Secure:  true,
	})
	return ""
}

// setUserDataCookie sets the cookie containing the users account data
func setUserSession(w http.ResponseWriter, token *oauth2.Token) {
	sessionID := uuid.NewString()
	sessionStore.Set(sessionID, token, cache.DefaultExpiration)

	http.SetCookie(w, &http.Cookie{
		Name:    "summit_userinfo",
		Value:   sessionID,
		Path:    "/",
		Expires: time.Now().Add(24 * time.Hour * 30),
	})
}

// getUserSession retrieves the user data from the user session cookie
func getUserSession(sessionID string) (*oauth2.Token, bool) {
	if data, found := sessionStore.Get(sessionID); found {
		return data.(*oauth2.Token), true
	}
	return nil, false
}

// checkUserCookie checks the stored browser cookie and returns the users information or an error
func checkUserCookie(w http.ResponseWriter, r *http.Request) (*oauth2.Token, error) {
	cookie, err := r.Cookie("summit_userinfo")
	if err == nil {
		// Verify cookie session
		if token, found := getUserSession(cookie.Value); found {
			return token, nil
		}

		// If verification failed — clear the bad cookie
		http.SetCookie(w, &http.Cookie{
			Name:    "summit_userinfo",
			Value:   "",
			Path:    "/",
			Expires: time.Unix(0, 0),
		})
	}
	return nil, errors.New("no session found")
}

// deleteCookie deletes the specified HTTP cookie from local storage
func deleteCookie(w http.ResponseWriter, cookie *http.Cookie) {
	cookie.Value = "none"
	cookie.Path = "/"
	http.SetCookie(w, cookie)
}

// getUserGuilds returns the guild IDs that the user can currently manage
// and the guild IDs that the user has the correct permission to manage if the bot is added
func getUserManagedGuilds(w http.ResponseWriter, r *http.Request, ctx context.Context, token *oauth2.Token) (map[string]string, map[string]string) {
	user := tokenToUser(w, r, ctx, token)

	managedGuilds := make(map[string]string)
	for _, guild := range common.Session.State.Guilds {
		member, err := common.Session.GuildMember(guild.ID, user.ID)
		if err != nil {
			continue
		}
		managed := isUserManaged(guild.ID, member)
		if managed {
			// Store the guild ID and name in the map
			managedGuilds[guild.ID] = guild.Name
		}
	}

	client := OauthConf.Client(ctx, token)
	resp, err := client.Get("https://discord.com/api/v10/users/@me/guilds")
	if err != nil {
	}

	if resp.StatusCode != http.StatusOK {
	}
	defer resp.Body.Close()

	type PartialGuild struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Permissions string `json:"permissions"`
	}
	var guilds []PartialGuild
	json.NewDecoder(resp.Body).Decode(&guilds)

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

	return managedGuilds, availableGuilds
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

// validateGuild ensures users can't access the manage page for guilds without the correct permissions
func validateGuild(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		guildIDStr := pat.Param(r, "server")
		_, err := strconv.ParseInt(guildIDStr, 10, 64)
		if err != nil {
			http.Redirect(w, r, "/?error=invalid_guild", http.StatusFound)
			return
		}

		token, err := checkUserCookie(w, r)
		if err != nil {
			http.Redirect(w, r, "/?error=no_access", http.StatusFound)
			return
		}
		user := tokenToUser(w, r, r.Context(), token)

		member, err := functions.GetMember(guildIDStr, user.ID)
		if err != nil {
			http.Redirect(w, r, "/?error=no_access", http.StatusFound)
			return
		}

		managed := isUserManaged(guildIDStr, member)
		if !managed {
			http.Redirect(w, r, "/?error=no_access", http.StatusFound)
			return
		}

		inner.ServeHTTP(w, r)
	})
}

type TmplContextData map[string]interface{}

// baseTemplateDataMW provides the initial template data to be parsed within each page
func baseTemplateDataMW(inner http.Handler) http.Handler {
	middleware := func(w http.ResponseWriter, r *http.Request) {
		baseData := TmplContextData{
			"HomeURL": URL,
			"Year":    time.Now().UTC().Year(),
			"Path":    r.URL.Path,
			"Version": common.VERSION,
			"Testing": common.ConfigTestMode,
		}
		ctx := context.WithValue(r.Context(), CtxKeyTmplData, baseData)

		inner.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(middleware)
}

type GithubRelease struct {
	HTMLURL     string    `json:"html_url"`
	Name        string    `json:"name"`
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
	BodyHTML    template.HTML
}

// userAndManagedGuildsInfoMW provides middleware to parse the current user data and the list of manageable guilds to the template data
func userAndManagedGuildsInfoMW(inner http.Handler) http.Handler {
	middleware := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		token, err := checkUserCookie(w, r)
		if err != nil {
			inner.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		user := tokenToUser(w, r, ctx, token)

		managedGuilds, availableGuilds := getUserManagedGuilds(w, r, ctx, token)
		// Remove managed guilds from available list
		for id := range managedGuilds {
			delete(availableGuilds, id)
		}

		fullManagedGuilds := getPopulatedGuildList(managedGuilds, URL+"/static/img/icons/question.svg", true)
		fullAvailableGuilds := getPopulatedGuildList(availableGuilds, URL+"/static/img/icons/plus.svg", false)

		tmplData, _ := ctx.Value(CtxKeyTmplData).(TmplContextData)
		tmplData["User"] = user
		tmplData["ManagedGuilds"] = fullManagedGuilds
		tmplData["AvailableGuilds"] = fullAvailableGuilds

		ctx = context.WithValue(ctx, CtxKeyTmplData, tmplData)
		inner.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(middleware)
}

// currentGuildDataMW provides middleware to parse the current guilds data to the template data
func currentGuildDataMW(inner http.Handler) http.Handler {
	middleware := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		guildID := pat.Param(r, "server")

		retrievedGuild, _ := common.Session.Guild(guildID)

		channels, _ := common.Session.GuildChannels(guildID)
		sort.SliceStable(channels, func(i, j int) bool {
			return channels[i].Position < channels[j].Position
		})

		roles := retrievedGuild.Roles
		sort.SliceStable(roles, func(i, j int) bool {
			return roles[i].Position > roles[j].Position
		})

		member, _ := functions.GetMember(retrievedGuild.ID, common.Bot.ID)
		role := functions.HighestRole(retrievedGuild.ID, member)

		guildData := map[string]interface{}{
			"ID":                     retrievedGuild.ID,
			"Name":                   retrievedGuild.Name,
			"Avatar":                 retrievedGuild.IconURL("1024"),
			"Channels":               channels,
			"Roles":                  roles,
			"BotHighestRolePosition": role.Position,
		}
		if guildData["Avatar"] == "" {
			guildData["Avatar"] = URL + "/static/img/icons/question.svg"
		}

		tmplData, _ := ctx.Value(CtxKeyTmplData).(TmplContextData)
		tmplData["CurrentGuild"] = guildData

		ctx = context.WithValue(ctx, CtxKeyTmplData, tmplData)
		inner.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(middleware)
}

// urlDataMW provides middleware to parse the URL data to the template data
func urlDataMW(inner http.Handler) http.Handler {
	middleware := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		u, err := url.Parse(TermsURL)
		termsURL := URL + "/terms"
		if err == nil {
			termsURL = u.String()
		}

		u, err = url.Parse(PrivacyURL)
		privacyURL := URL + "/privacy"
		if err == nil {
			privacyURL = u.String()
		}

		tmplData, _ := ctx.Value(CtxKeyTmplData).(TmplContextData)
		tmplData["TermsURL"] = termsURL
		tmplData["PrivacyURL"] = privacyURL

		ctx = context.WithValue(ctx, CtxKeyTmplData, tmplData)
		inner.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(middleware)
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

// tokenToUser converts an OAuth token to a valid discordgo.User object
func tokenToUser(w http.ResponseWriter, r *http.Request, ctx context.Context, token *oauth2.Token) *discordgo.User {
	client := OauthConf.Client(ctx, token)
	resp, err := client.Get("https://discord.com/api/v10/users/@me")
	if err != nil {
		http.Redirect(w, r, "/?error=failed_retrieving_info", http.StatusTemporaryRedirect)
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		http.Redirect(w, r, "/?error=discord_api_error", http.StatusTemporaryRedirect)
		return nil
	}
	defer resp.Body.Close()

	var user *discordgo.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		http.Redirect(w, r, "/?error=json_decode_error", http.StatusTemporaryRedirect)
		return nil
	}

	return user
}

func GetForm[T any](r *http.Request) *T {
	if v, ok := r.Context().Value(CtxKeyFormParsed).(*T); ok {
		return v
	}
	return nil
}

// ParseForm parses the request into the provided form struct, validates
// field with a `valid` tag and stores the result in the request context.
// The parsed form can be accesed via web.CtxKeyFormParsed if validation succeeds.
//
// The form parameter should be a struct or a pointer to a struct with fields matching HTML inputs.
//
// # ParseForm does not save forms, and that must be done manually through a config handler
//
// Fields not present in the form (e.g guild IDs, or internal config values)
// are NOT automatically populated and MUST be set manually.
//
// Example:
//
//	oldCfg := GetConfig(guild)
//	newCfg.GuildID = oldCfg.GuildID
func ParseForm(inner http.Handler, form interface{}) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		guildID := pat.Param(r, "server")
		guild := functions.GetGuild(guildID)

		// Ensure we retrieve the underlying type instead of potentially having a pointer-to-pointer
		formType := reflect.TypeOf(form)
		if formType.Kind() == reflect.Ptr {
			formType = formType.Elem()
		}

		// Decode the sent form into the struct
		decoded := reflect.New(formType).Interface()
		decoder := schema.NewDecoder()
		decoder.IgnoreUnknownKeys(true)

		if err := decoder.Decode(decoded, r.PostForm); err != nil {
			SendErrorToast(w, fmt.Sprintf("Failed to decode form: %s", err.Error()))
			return
		}

		if err := validateForm(guild, decoded); err != nil {
			SendErrorToast(w, err.Error())
			return
		}

		// Add to context
		ctx := context.WithValue(r.Context(), CtxKeyFormParsed, decoded)
		inner.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(handler)
}
