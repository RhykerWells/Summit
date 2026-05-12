package moderation

import (
	"context"
	"embed"
	"net/http"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/common"
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

	moderationMux.Handle(pat.Post(""), web.ParseForm(http.HandlerFunc(saveConfigHandler), Config{}))
	moderationMux.Handle(pat.Post("/"), web.ParseForm(http.HandlerFunc(saveConfigHandler), Config{}))

	moderationMux.HandleFunc(pat.Get("/cases"), web.RenderPage("cases.html"))
	moderationMux.HandleFunc(pat.Get("/cases/"), web.RenderPage("cases.html"))

	moderationMux.HandleFunc(pat.Get("/logs"), web.RenderPage("logs.html"))
	moderationMux.HandleFunc(pat.Get("/logs/"), web.RenderPage("logs.html"))

	moderationMux.HandleFunc(pat.Get("/logs/:id"), handleMessageLogs)
	moderationMux.HandleFunc(pat.Get("/logs/:id/"), handleMessageLogs)
}

// saveConfigHandler saves the parsed form data and saves it if possible.
func saveConfigHandler(w http.ResponseWriter, r *http.Request) {
	newCfg := web.GetForm[Config](r)

	guildID := pat.Param(r, "server")
	oldCfg := GetConfig(guildID)

	// Ensure these non-editable fields are still present in the new form
	newCfg.GuildID = oldCfg.GuildID
	newCfg.LastCaseID = oldCfg.LastCaseID

	err := SaveConfig(newCfg)
	if err != nil {
		web.SendErrorToast(w, err.Error())
		return
	}

	web.SendSuccessToast(w, "Successfully saved")
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

		messageLogs := getGuildMessageLogs(guildID)
		tmplData["MessageLogs"] = messageLogs

		ctx = context.WithValue(ctx, web.CtxKeyTmplData, tmplData)
		inner.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(middleware)
}

func handleMessageLogs(w http.ResponseWriter, r *http.Request) {
	logID := pat.Param(r, "id")
	logIDInt64 := functions.ToInt64(logID)

	log, err := getMessageLogByID(logIDInt64)
	if err != nil {
		web.SendErrorToast(w, "Message log not found")
		return
	}

	log.L.LoadLogModerationMessageLogsMessages(context.Background(), common.PQ, true, log, nil)

	tmplData, _ := r.Context().Value(web.CtxKeyTmplData).(web.TmplContextData)
	tmplData["MessageLog"] = log

	web.RenderPage("log.html")(w, r)
}
