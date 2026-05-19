package moderation

import (
	"context"
	"embed"
	"net/http"
	"sort"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/moderation/models"
	"github.com/RhykerWells/Summit/web"
	"goji.io/v3"
	"goji.io/v3/pat"
)

//go:embed assets/*
var PageHTML embed.FS

func initWeb() {
	web.AddHTMLFilesystem(PageHTML)
	web.RegisterDashboardRoutes(registerModerationRoutes)

	categoryModeration := web.SidebarCategory{
		Name: "Moderation",
		Icon: "fa-solid fa-users-gear",
		Items: []*web.SidebarItem{
			{
				Name: "Moderation",
				Icon: "fa-solid fa-users-gear",
				URL:  "moderation",
			},
			{
				Name: "Cases",
				Icon: "fa-solid fa-rectangle-list",
				URL:  "moderation/cases",
			},
			{
				Name: "Message Logs",
				Icon: "fa-solid fa-square-poll-horizontal",
				URL:  "moderation/logs",
			},
		},
	}

	web.AddSidebarCategory(categoryModeration)
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

		// Create a map of CaseID to MessageLogID for linking
		caseLogMap := make(map[int64]int64)
		for _, log := range messageLogs {
			if log.CaseID != 0 {
				caseLogMap[log.CaseID] = log.ID
			}
		}
		tmplData["CaseLogMap"] = caseLogMap

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

	// Sort messages by creation date (ascending - oldest first)
	sort.Slice(log.R.LogModerationMessageLogsMessages, func(i, j int) bool {
		return log.R.LogModerationMessageLogsMessages[i].CreatedAt.Before(log.R.LogModerationMessageLogsMessages[j].CreatedAt)
	})

	tmplData, _ := r.Context().Value(web.CtxKeyTmplData).(web.TmplContextData)

	caseData, err := models.FindModerationCaseG(context.Background(), log.GuildID, log.CaseID)
	if err == nil {
		tmplData["MessageCase"] = caseData
	}

	tmplData["MessageLog"] = log

	web.RenderPage("log.html")(w, r)
}
