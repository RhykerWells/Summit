package economy

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/commands/economy/models"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/web"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/gorilla/schema"
	"goji.io/v3"
	"goji.io/v3/pat"
)

//go:embed assets/*
var PageHTML embed.FS

func initWeb() {
	web.AddHTMLFilesystem(PageHTML)
	web.RegisterDashboardRoutes(registerEconomyRoutes)
}

func registerEconomyRoutes(dashboard *goji.Mux) {
	economyMux := goji.SubMux()

	economyMux.Use(economyMW)

	dashboard.Handle(pat.New("/economy"), economyMux)
	dashboard.Handle(pat.New("/economy/*"), economyMux)

	economyMux.HandleFunc(pat.Get(""), web.RenderPage("economy.html"))
	economyMux.HandleFunc(pat.Get("/"), web.RenderPage("economy.html"))

	economyMux.Handle(pat.Post(""), web.ParseForm(http.HandlerFunc(saveConfigHandler), Config{}))
	economyMux.Handle(pat.Post("/"), web.ParseForm(http.HandlerFunc(saveConfigHandler), Config{}))

	economyMux.HandleFunc(pat.Post("/responses/:type/new"), saveNewResponseHandler)
	economyMux.HandleFunc(pat.Post("/responses/:type/new/"), saveNewResponseHandler)

	economyMux.HandleFunc(pat.Post("/responses/:type/:id/edit"), editResponseHandler)
	economyMux.HandleFunc(pat.Post("/responses/:type/:id/edit/"), editResponseHandler)

	economyMux.HandleFunc(pat.Post("/responses/:type/:id/delete"), deleteResponseHandler)
	economyMux.HandleFunc(pat.Post("/responses/:type/:id/delete/"), deleteResponseHandler)

	economyMux.HandleFunc(pat.Get("/shop"), web.RenderPage("shop.html"))
	economyMux.HandleFunc(pat.Get("/shop/"), web.RenderPage("shop.html"))

	economyMux.HandleFunc(pat.Post("/shop"), saveItemHandler)
	economyMux.HandleFunc(pat.Post("/shop/"), saveItemHandler)
}

// economyMW provides middleware to parse all the economy data to the template data
func economyMW(inner http.Handler) http.Handler {
	middleware := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		guildID := pat.Param(r, "server")

		config := GetConfig(guildID)

		tmplData, _ := ctx.Value(web.CtxKeyTmplData).(web.TmplContextData)
		tmplData["EconomyConfig"] = config

		workResponses, _ := models.EconomyResponses(models.EconomyResponseWhere.GuildID.EQ(guildID), models.EconomyResponseWhere.Type.EQ("work")).All(ctx, common.PQ)
		tmplData["WorkResponses"] = workResponses

		crimeResponses, _ := models.EconomyResponses(models.EconomyResponseWhere.GuildID.EQ(guildID), models.EconomyResponseWhere.Type.EQ("crime")).All(ctx, common.PQ)
		tmplData["CrimeResponses"] = crimeResponses

		store := getGuildShop(guildID)
		tmplData["Store"] = store

		ctx = context.WithValue(ctx, web.CtxKeyTmplData, tmplData)
		inner.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(middleware)
}

func saveConfigHandler(w http.ResponseWriter, r *http.Request) {
	guildID := pat.Param(r, "server")
	oldCfg := GetConfig(guildID)

	newCfg := web.GetForm[Config](r)
	// Ensure these non-editable fields are still present in the new form
	newCfg.GuildID = oldCfg.GuildID
	newCfg.EconomyCustomWorkResponses = oldCfg.EconomyCustomWorkResponses
	newCfg.EconomyCustomCrimeResponses = oldCfg.EconomyCustomCrimeResponses

	err := SaveConfig(newCfg)
	if err != nil {
		web.SendErrorToast(w, err.Error())
		return
	}

	web.SendSuccessToast(w, "Successfully saved")
}

func saveNewResponseHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	guildID := pat.Param(r, "server")
	guild := functions.GetGuild(guildID)

	responseType := pat.Param(r, "type")
	if responseType != "work" && responseType != "crime" {
		web.SendErrorToast(w, "Invalid response type.")
		return
	}

	var response models.EconomyResponse

	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err := decoder.Decode(&response, r.PostForm)
	if err != nil {
		web.SendErrorToast(w, fmt.Sprintf("Failed to decode form: %s", err.Error()))
		return
	}

	response.GuildID = guild.ID
	response.Type = responseType

	// Validate the response contains the literal string (amount)
	match, err := regexp.MatchString(`\(amount\)`, response.Response)
	if err != nil {
		web.SendErrorToast(w, fmt.Sprintf("Failed to validate response: %s", err.Error()))
		return
	}
	if !match {
		web.SendErrorToast(w, "Response must contain the literal string (amount).")
		return
	}

	err = response.Insert(context.Background(), common.PQ, boil.Infer())
	if err != nil {
		web.SendErrorToast(w, fmt.Sprintf("Failed to save response: %s", err.Error()))
		return
	}

	web.SendSuccessToast(w, "Successfully saved.")
}

func editResponseHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	guildID := pat.Param(r, "server")
	guild := functions.GetGuild(guildID)

	responseType := pat.Param(r, "type")
	if responseType != "work" && responseType != "crime" {
		web.SendErrorToast(w, "Invalid response type.")
		return
	}

	responseID := pat.Param(r, "id")
	responseIDInt, _ := strconv.Atoi(responseID)

	responseEntry, err := models.EconomyResponses(models.EconomyResponseWhere.GuildID.EQ(guild.ID), models.EconomyResponseWhere.ID.EQ(responseIDInt)).One(context.Background(), common.PQ)
	if err != nil {
		web.SendErrorToast(w, "Response not found.")
		return
	}

	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err = decoder.Decode(&responseEntry, r.PostForm)
	if err != nil {
		web.SendErrorToast(w, fmt.Sprintf("Failed to decode form: %s", err.Error()))
		return
	}

	// Validate the response contains the literal string (amount)
	match, err := regexp.MatchString(`\(amount\)`, responseEntry.Response)
	if err != nil {
		web.SendErrorToast(w, fmt.Sprintf("Failed to validate response: %s", err.Error()))
		return
	}
	if !match {
		web.SendErrorToast(w, "Response must contain the literal string (amount).")
		return
	}

	responseEntry.Update(context.Background(), common.PQ, boil.Infer())
	web.SendSuccessToast(w, "Successfully saved.")
}

func deleteResponseHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	guildID := pat.Param(r, "server")
	guild := functions.GetGuild(guildID)

	responseType := pat.Param(r, "type")
	if responseType != "work" && responseType != "crime" {
		web.SendErrorToast(w, "Invalid response type.")
		return
	}

	responseID := pat.Param(r, "id")
	responseIDInt, _ := strconv.Atoi(responseID)

	responseEntry, err := models.EconomyResponses(models.EconomyResponseWhere.GuildID.EQ(guild.ID), models.EconomyResponseWhere.ID.EQ(responseIDInt)).One(context.Background(), common.PQ)
	if err != nil {
		web.SendErrorToast(w, "Response not found.")
		return
	}

	responseEntry.Delete(context.Background(), common.PQ)
	web.SendSuccessToast(w, "Successfully deleted.")
}

func saveItemHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	r.ParseForm()

	guildID := pat.Param(r, "server")

	item, _ := models.EconomyShops(models.EconomyShopWhere.GuildID.EQ(guildID), models.EconomyShopWhere.Name.EQ(r.FormValue("item"))).One(context.Background(), common.PQ)
	index := r.FormValue("index")

	var err error

	formType := r.FormValue("form_type")
	switch formType {
	case "editItem":
		htmlInput := fmt.Sprintf("editItem%s", index)

		name := r.FormValue(htmlInput + "Name")
		description := r.FormValue(htmlInput + "Description")
		price := r.FormValue(htmlInput + "Price")
		quantity := r.FormValue(htmlInput + "Quantity")
		role := r.FormValue(htmlInput + "Role")
		reply := r.FormValue(htmlInput + "Reply")

		if r.FormValue("item") != name {
			if !newItemNameOk(w, guildID, name) {
				return
			}
		}

		nameMaxLength, _ := strconv.Atoi(r.FormValue(htmlInput + "NameMaxLength"))
		if len(name) > nameMaxLength {
			web.SendErrorToast(w, fmt.Sprintf("The name must be less than %d characters.", nameMaxLength))
			return
		}

		descriptionMaxLength, _ := strconv.Atoi(r.FormValue(htmlInput + "NameMaxLength"))
		if len(description) > descriptionMaxLength {
			web.SendErrorToast(w, fmt.Sprintf("The description must be less than %d characters.", descriptionMaxLength))
			return
		}

		replyMaxLength, _ := strconv.Atoi(r.FormValue(htmlInput + "ReplyMaxLength"))
		if len(reply) > replyMaxLength {
			web.SendErrorToast(w, fmt.Sprintf("The reply must be less than %d characters.", replyMaxLength))
			return
		}

		item.Description = description
		item.Price = functions.ToInt64(price)
		item.Quantity = functions.ToInt64(quantity)

		// Replace empty role ID via modification
		if _, err := functions.GetRole(guildID, role); err != nil {
			role = ""
		}
		item.Role = role

		item.Reply = reply

		if name != r.FormValue("item") {
			_, err := common.PQ.ExecContext(context.Background(), `UPDATE economy_shop SET name = $1 WHERE guild_id = $2 AND name = $3`, name, guildID, r.FormValue("item"))
			if err != nil {
				web.SendErrorToast(w, err.Error())
				return
			}
		}

		_, err = item.Update(context.Background(), common.PQ, boil.Infer())
		if err == nil {
			item.Reload(context.Background(), common.PQ)
		}
	case "deleteItem":
		item.Delete(context.Background(), common.PQ)
	case "newItem":
		htmlInput := "newItem"

		name := r.FormValue(htmlInput + "Name")
		description := r.FormValue(htmlInput + "Description")
		price := r.FormValue(htmlInput + "Price")
		quantity := r.FormValue(htmlInput + "Quantity")
		role := r.FormValue(htmlInput + "Role")
		reply := r.FormValue(htmlInput + "Reply")
		if r.FormValue("item") != name {
			if !newItemNameOk(w, guildID, name) {
				return
			}
		}

		nameMaxLength, _ := strconv.Atoi(r.FormValue(htmlInput + "NameMaxLength"))
		if len(name) > nameMaxLength {
			web.SendErrorToast(w, fmt.Sprintf("The name must be less than %d characters.", nameMaxLength))
			return
		}

		// Replace empty role ID via creation
		if _, err := functions.GetRole(guildID, role); err != nil {
			role = ""
		}

		item := models.EconomyShop{
			GuildID:     guildID,
			Name:        name,
			Description: description,
			Price:       functions.ToInt64(price),
			Quantity:    functions.ToInt64(quantity),
			Role:        role,
			Reply:       reply,
		}
		err = item.Insert(context.Background(), common.PQ, boil.Infer())
	}

	if err != nil {
		web.SendErrorToast(w, err.Error())
		return
	}

	web.SendSuccessToast(w, "Successfully saved")
}

func parseCustomResponse(w http.ResponseWriter, r *http.Request, fieldName string) (string, error) {
	response := r.FormValue(fieldName)

	re := regexp.MustCompile(`\(amount\)`)
	match := re.MatchString(response)

	if !match {
		return "", errors.New("Response did not contain literal string <code style=\"color: white;\">(amount)</code>")
	}
	return response, nil
}

func newItemNameOk(w http.ResponseWriter, guildID string, newName string) bool {
	currentItem, _ := models.EconomyShops(models.EconomyShopWhere.GuildID.EQ(guildID), models.EconomyShopWhere.Name.EQ(newName)).One(context.Background(), common.PQ)
	if currentItem != nil {
		web.SendErrorToast(w, "Item with this name already exists.")
		return false
	}

	return true
}
