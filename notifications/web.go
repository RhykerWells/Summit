package notifications

import (
	"context"
	"embed"
	"net/http"

	"github.com/RhykerWells/Summit/web"
	"goji.io/v3"
	"goji.io/v3/pat"
)

//go:embed assets/*
var PageHTML embed.FS

func initWeb() {
	web.AddHTMLFilesystem(PageHTML)
	web.RegisterDashboardRoutes(registerNotificationRoutes)

	categoryNotifications := web.SidebarCategory{
		Name: "Notifications",
		Icon: "fa-solid fa-bell",
		Items: []*web.SidebarItem{
			{
				Name: "Notifications",
				Icon: "fa-solid fa-bell",
				URL:  "notifications",
			},
		},
	}

	web.AddSidebarCategory(categoryNotifications)
}

func registerNotificationRoutes(dashboard *goji.Mux) {
	notificationMux := goji.SubMux()

	notificationMux.Use(notificationMW)

	dashboard.Handle(pat.New("/notifications"), notificationMux)
	dashboard.Handle(pat.New("/notifications/*"), notificationMux)

	notificationMux.HandleFunc(pat.Get(""), web.RenderPage("notifications.html"))
	notificationMux.HandleFunc(pat.Get("/"), web.RenderPage("notifications.html"))

	notificationMux.Handle(pat.Post(""), web.ParseForm(http.HandlerFunc(saveConfigHandler), Config{}))
	notificationMux.Handle(pat.Post("/"), web.ParseForm(http.HandlerFunc(saveConfigHandler), Config{}))
}

// notificationMW provides middleware to parse all the notification data to the template data
func notificationMW(inner http.Handler) http.Handler {
	middleware := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		guildID := pat.Param(r, "server")

		config := GetConfig(guildID)

		tmplData, _ := ctx.Value(web.CtxKeyTmplData).(web.TmplContextData)
		tmplData["NotificationConfig"] = config

		ctx = context.WithValue(ctx, web.CtxKeyTmplData, tmplData)
		inner.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(middleware)
}

// saveConfigHandler parses form data and saves it if possible.
func saveConfigHandler(w http.ResponseWriter, r *http.Request) {
	newCfg := web.GetForm[Config](r)

	guildID := pat.Param(r, "server")
	oldCfg := GetConfig(guildID)

	// Ensure these non-editable fields are still present in the new form
	newCfg.GuildID = oldCfg.GuildID

	err := SaveConfig(newCfg)
	if err != nil {
		web.SendErrorToast(w, err.Error())
		return
	}

	web.SendSuccessToast(w, "Successfully saved")
}
