package economy

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/commands/economy/models"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/web"
	"github.com/aarondl/sqlboiler/v4/boil"
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

	economyMux.HandleFunc(pat.Post(""), saveConfigHandler)
	economyMux.HandleFunc(pat.Post("/"), saveConfigHandler)

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

		store := getGuildShop(guildID)
		tmplData["Store"] = store

		ctx = context.WithValue(ctx, web.CtxKeyTmplData, tmplData)
		inner.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(middleware)
}

func saveConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	r.ParseForm()

	guildID := pat.Param(r, "server")
	config := GetConfig(guildID)

	formType := r.FormValue("form_type")
	switch formType {
	case "Core":
		economyEnabledBool, _ := strconv.ParseBool(r.FormValue("EconomyEnabled"))
		economyStartbalance, _ := strconv.ParseInt(r.FormValue("EconomyStartBalance"), 10, 64)
		economyMinimumReturn, _ := strconv.ParseInt(r.FormValue("EconomyMinReturn"), 10, 64)
		economyMaximumReturn, _ := strconv.ParseInt(r.FormValue("EconomyMaxReturn"), 10, 64)
		economyMaximumBet, _ := strconv.ParseInt(r.FormValue("EconomyMaxBet"), 10, 64)
		economyCustomWorkResponsesEnabled, _ := strconv.ParseBool(r.FormValue("EconomyCustomWorkResponsesEnabled"))
		economyCustomCrimeResponsesEnabled, _ := strconv.ParseBool(r.FormValue("EconomyCustomCrimeResponsesEnabled"))

		config.EconomyEnabled = economyEnabledBool
		config.EconomySymbol = r.FormValue("EconomySymbol")
		config.EconomyStartBalance = economyStartbalance
		config.EconomyMinReturn = economyMinimumReturn
		config.EconomyMaxReturn = economyMaximumReturn
		config.EconomyMaxBet = economyMaximumBet
		config.EconomyCustomWorkResponsesEnabled = economyCustomWorkResponsesEnabled
		config.EconomyCustomCrimeResponsesEnabled = economyCustomCrimeResponsesEnabled
	case "newWorkResponse":
		response, ok := parseCustomResponse(w, r, "workResponse")
		if !ok {
			return
		}
		config.EconomyCustomWorkResponses = append(config.EconomyCustomWorkResponses, response)
	case "editWorkResponse":
		index := functions.ToInt64(r.FormValue("index"))
		formKey := fmt.Sprintf("%dWorkResponse", index)

		if index < 0 || int(index) >= len(config.EconomyCustomWorkResponses) {
			web.SendErrorToast(w, "This response doesn't exist/")
			return
		}

		response, ok := parseCustomResponse(w, r, formKey)
		if !ok {
			return
		}
		config.EconomyCustomWorkResponses[index] = response
	case "deleteWorkResponse":
		index := functions.ToInt64(r.FormValue("index"))

		config.EconomyCustomWorkResponses = append(
			config.EconomyCustomWorkResponses[:index],
			config.EconomyCustomWorkResponses[index+1:]...,
		)
	case "newCrimeResponse":
		response, ok := parseCustomResponse(w, r, "crimeResponse")
		if !ok {
			return
		}
		config.EconomyCustomCrimeResponses = append(config.EconomyCustomCrimeResponses, response)
	case "editCrimeResponse":
		index := functions.ToInt64(r.FormValue("index"))
		formKey := fmt.Sprintf("%dCrimeResponse", index)

		if index < 0 || int(index) >= len(config.EconomyCustomWorkResponses) {
			web.SendErrorToast(w, "This response doesn't exist/")
			return
		}

		response, ok := parseCustomResponse(w, r, formKey)
		if !ok {
			return
		}
		config.EconomyCustomWorkResponses[index] = response
	case "deleteCrimeResponse":
		index := functions.ToInt64(r.FormValue("index"))

		config.EconomyCustomCrimeResponses = append(
			config.EconomyCustomCrimeResponses[:index],
			config.EconomyCustomCrimeResponses[index+1:]...,
		)
	}

	err := SaveConfig(config)
	if err != nil {
		web.SendErrorToast(w, err.Error())
		return
	}

	web.SendSuccessToast(w, "Successfully saved")
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

func parseCustomResponse(w http.ResponseWriter, r *http.Request, fieldName string) (string, bool) {
	response := r.FormValue(fieldName)

	re := regexp.MustCompile(`\(amount\)`)
	match := re.MatchString(response)

	if !match {
		web.SendErrorToast(w, "Response did not contain literal string <code style=\"color: white;\">(amount)</code>")
		return "", false
	}
	return response, true
}

func newItemNameOk(w http.ResponseWriter, guildID string, newName string) bool {
	currentItem, _ := models.EconomyShops(models.EconomyShopWhere.GuildID.EQ(guildID), models.EconomyShopWhere.Name.EQ(newName)).One(context.Background(), common.PQ)
	if currentItem != nil {
		web.SendErrorToast(w, "Item with this name already exists.")
		return false
	}
	return true
}
