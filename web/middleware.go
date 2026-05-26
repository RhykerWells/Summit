package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"reflect"
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
	CtxKeyCurrentUser
	CtxKeyOAuthToken
	CtxKeyUserManagedGuilds
	CtxKeyAvailableGuilds
	CtxKeyCurrentGuild
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
		Name:     "summit_csrf",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(300 * time.Second),
		Secure:   true,
		HttpOnly: true,
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
		Name:     "summit_csrf",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		Secure:   true,
		HttpOnly: true,
	})
	return ""
}

// setUserDataCookie sets the cookie containing the users account data
func setUserSession(w http.ResponseWriter, token *oauth2.Token) {
	sessionID := uuid.NewString()
	sessionStore.Set(sessionID, token, cache.DefaultExpiration)

	http.SetCookie(w, &http.Cookie{
		Name:     "summit_userinfo",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour * 30),
		Secure:   true,
		HttpOnly: true,
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
			Name:     "summit_userinfo",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			Secure:   true,
			HttpOnly: true,
		})
	}
	return nil, errors.New("no session found")
}

// deleteCookie deletes the specified HTTP cookie from local storage
func deleteCookie(w http.ResponseWriter, cookie *http.Cookie) {
	cookie.Value = "none"
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.Secure = true
	http.SetCookie(w, cookie)
}

type GithubRelease struct {
	HTMLURL     string    `json:"html_url"`
	Name        string    `json:"name"`
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
	BodyHTML    template.HTML
}

type TmplContextData map[string]interface{}

func BaseTemplateDataMW(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		baseData := TmplContextData{
			"HomeURL": URL,
			"Year":    time.Now().UTC().Year(),
			"Path":    r.URL.Path,
			"Version": common.VERSION,
			"Testing": common.ConfigTestMode,
			"Sidebar": sidebarData,
		}

		inner.ServeHTTP(w, r.WithContext(SetTmplContextData(r.Context(), baseData)))
	})
}

func CurrentUserMW(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		token, err := checkUserCookie(w, r)
		if err != nil {
			inner.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		user, err := tokenToUser(ctx, token)
		if err != nil {
			inner.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		managedGuilds, availableGuilds, err := getUserManagedGuilds(ctx, token)
		if err != nil {
			inner.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		ctx = context.WithValue(ctx, CtxKeyCurrentUser, user)
		ctx = context.WithValue(ctx, CtxKeyOAuthToken, token)
		ctx = context.WithValue(ctx, CtxKeyUserManagedGuilds, managedGuilds)
		ctx = context.WithValue(ctx, CtxKeyAvailableGuilds, availableGuilds)

		tmplData := TmplContextData{
			"User":            user,
			"ManagedGuilds":   managedGuilds,
			"AvailableGuilds": availableGuilds,
		}

		inner.ServeHTTP(w, r.WithContext(SetTmplContextData(ctx, tmplData)))
	})
}

func RequireUserMW(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		user, ok := ctx.Value(CtxKeyCurrentUser).(*discordgo.User)
		if !ok || user == nil {
			http.Redirect(w, r, "/?error=no_access", http.StatusTemporaryRedirect)
			return
		}

		inner.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CurrentGuildMW(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		guildID := pat.Param(r, "server")

		guild, err := common.Session.Guild(guildID)
		if err != nil {
			http.Redirect(w, r, "/?error=invalid_guild", http.StatusNotFound)
			return
		}

		user, ok := ctx.Value(CtxKeyCurrentUser).(*discordgo.User)
		if !ok || user == nil {
			http.Redirect(w, r, "/?error=no_access", http.StatusTemporaryRedirect)
			return
		}

		member, err := functions.GetMember(guildID, user.ID)
		if err != nil {
			http.Redirect(w, r, "/?error=no_access", http.StatusTemporaryRedirect)
			return
		}

		if !isUserManaged(guildID, member) {
			http.Redirect(w, r, "/?error=no_access", http.StatusTemporaryRedirect)
			return
		}

		tmplData := TmplContextData{
			"CurrentGuild": getGuildTmplData(guild),
		}

		ctx = context.WithValue(ctx, CtxKeyCurrentGuild, guild)

		inner.ServeHTTP(w, r.WithContext(SetTmplContextData(ctx, tmplData)))
	})
}

func RequireGuildMW(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		guild, ok := ctx.Value(CtxKeyCurrentGuild).(*discordgo.Guild)
		if !ok || guild == nil {
			http.Redirect(w, r, "/?error=no_access", http.StatusTemporaryRedirect)
			return
		}

		inner.ServeHTTP(w, r.WithContext(ctx))
	})
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
