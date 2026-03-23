package moderation

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/RhykerWells/Summit/web"
	"goji.io/v3"
	"goji.io/v3/pat"
)

//go:embed assets/*
var PageHTML embed.FS

func initWeb() {
	web.AddHTMLFilesystem(PageHTML)
	web.RegisterDashboardRoutes(registerModerationRoutes)
}

func registerModerationRoutes(dashboard *goji.Mux) {
	moderationMux := goji.SubMux()

	moderationMux.Use(moderationMW)

	dashboard.Handle(pat.New("/moderation"), moderationMux)
	dashboard.Handle(pat.New("/moderation/*"), moderationMux)

	moderationMux.HandleFunc(pat.Get(""), web.RenderPage("moderation.html"))
	moderationMux.HandleFunc(pat.Get("/"), web.RenderPage("moderation.html"))

	moderationMux.HandleFunc(pat.Post(""), saveConfigHandler)
	moderationMux.HandleFunc(pat.Post("/"), saveConfigHandler)

	moderationMux.HandleFunc(pat.Get("/cases"), web.RenderPage("cases.html"))
	moderationMux.HandleFunc(pat.Get("/cases/"), web.RenderPage("cases.html"))
}

func saveConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	r.ParseForm()

	guildID := pat.Param(r, "server")
	config := GetConfig(guildID)

	formType := r.FormValue("form_type")
	switch formType {
	case "Core":
		moderationEnabledBool, _ := strconv.ParseBool(r.FormValue("ModerationEnabled"))
		moderationTriggerDeletionEnabled, _ := strconv.ParseBool(r.FormValue("ModerationTriggerDeletionEnabled"))
		moderationTriggerDeletionSeconds, _ := strconv.ParseInt(r.FormValue("ModerationTriggerDeletionSeconds"), 10, 64)
		moderationResponseDeletionEnabled, _ := strconv.ParseBool(r.FormValue("ModerationResponseDeletionEnabled"))
		moderationResponseDeletionSeconds, _ := strconv.ParseInt(r.FormValue("ModerationResponseDeletionSeconds"), 10, 64)

		config.ModerationEnabled = moderationEnabledBool
		config.ModerationLogChannel = r.FormValue("ModerationLogChannel")
		config.ModerationTriggerDeletionEnabled = moderationTriggerDeletionEnabled
		config.ModerationTriggerDeletionSeconds = moderationTriggerDeletionSeconds
		config.ModerationResponseDeletionEnabled = moderationResponseDeletionEnabled
		config.ModerationResponseDeletionSeconds = moderationResponseDeletionSeconds
	case "Warn":
		roleArray, ok := parseRoleArray(w, r, "WarnRequiredRoles")
		if !ok {
			return
		}
		config.WarnRequiredRoles = roleArray
	case "Mute":
		roleArray, ok := parseRoleArray(w, r, "MuteRequiredRoles")
		if !ok {
			return
		}
		config.MuteRequiredRoles = roleArray

		config.MuteRole = r.FormValue("MuteRole")
		config.MuteManageRole = r.FormValue("MuteManageRole") == "true"

		roleArray, ok = parseRoleArray(w, r, "MuteUpdateRoles")
		if !ok {
			return
		}
		config.MuteUpdateRoles = roleArray
	case "Kick":
		roleArray, ok := parseRoleArray(w, r, "KickRequiredRoles")
		if !ok {
			return
		}
		config.KickRequiredRoles = roleArray
	case "Ban":
		roleArray, ok := parseRoleArray(w, r, "BanRequiredRoles")
		if !ok {
			return
		}
		config.BanRequiredRoles = roleArray
	}

	err := SaveConfig(config)
	if err != nil {
		web.SendErrorToast(w, err.Error())
		return
	}

	web.SendSuccessToast(w, "Successfully saved")
}

func parseRoleArray(w http.ResponseWriter, r *http.Request, fieldName string) ([]string, bool) {
	value := r.FormValue(fieldName)
	roleArray, err := unmarshalJsonArrayToGoArray(value)
	if err != nil {
		web.SendErrorToast(w, fmt.Sprintf("Invalid JSON data for %s", fieldName))
		return nil, false
	}
	return roleArray, true
}

func unmarshalJsonArrayToGoArray(jsonStr string) ([]string, error) {
	var result []string
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// moderationMW provides middleware to parse all the moderation data to the template data
func moderationMW(inner http.Handler) http.Handler {
	middleware := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		guildID := pat.Param(r, "server")

		config := GetConfig(guildID)

		tmplData, _ := ctx.Value(web.CtxKeyTmplData).(web.TmplContextData)
		tmplData["ModerationConfig"] = config

		cases := getGuildCases(guildID)
		tmplData["Cases"] = cases

		ctx = context.WithValue(ctx, web.CtxKeyTmplData, tmplData)
		inner.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(middleware)
}
