package core

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

// initWeb adds the specified routes the list of web routes for the web package to initialise
func initWeb() {
	web.AddHTMLFilesystem(PageHTML)
	web.RegisterDashboardRoutes(registerCoreRoute)
}

// registerCoreRoute initialises the core web routes
// and attaches the required middlewares for validation and template data
func registerCoreRoute(dashboard *goji.Mux) {
	// Create a sub-mux for core-related routes
	coreMux := goji.SubMux()

	// Middlewares
	coreMux.Use(coreConfigMW)

	// Server core pages
	dashboard.Handle(pat.New("/core"), coreMux)
	dashboard.Handle(pat.New("/core/"), coreMux)

	coreMux.HandleFunc(pat.Get(""), web.RenderPage("core.html"))
	coreMux.HandleFunc(pat.Get("/"), web.RenderPage("core.html"))

	// Data saving routes
	coreMux.Handle(pat.Post(""), web.ParseForm(http.HandlerFunc(saveConfigHandler), Config{}))
	coreMux.Handle(pat.Post("/"), web.ParseForm(http.HandlerFunc(saveConfigHandler), Config{}))
}

// saveConfigHandler saves the parsed form data and saves it if possible.
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

// coreConfigMW provides middleware to parse the core config data to the template data
func coreConfigMW(inner http.Handler) http.Handler {
	middleware := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		guildID := pat.Param(r, "server")

		config := GetConfig(guildID)

		tmplData, _ := ctx.Value(web.CtxKeyTmplData).(web.TmplContextData)
		tmplData["CoreConfig"] = config

		ctx = context.WithValue(ctx, web.CtxKeyTmplData, tmplData)
		inner.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(middleware)
}
