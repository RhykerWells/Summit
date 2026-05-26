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
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/economy/models"
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

	categoryEconomy := web.SidebarCategory{
		Name: "Economy",
		Icon: "fa-solid fa-coins",
		Items: []*web.SidebarItem{
			{
				Name: "Economy",
				Icon: "fa-solid fa-coins",
				URL:  "economy",
			},
			{
				Name: "Shop",
				Icon: "fa-solid fa-store",
				URL:  "economy/shop",
			},
		},
	}

	web.AddSidebarCategory(categoryEconomy)
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

	economyMux.HandleFunc(pat.Post("/shop/new"), saveNewItemHandler)
	economyMux.HandleFunc(pat.Post("/shop/new/"), saveNewItemHandler)

	economyMux.HandleFunc(pat.Post("/shop/:id/edit"), editItemHandler)
	economyMux.HandleFunc(pat.Post("/shop/:id/edit/"), editItemHandler)

	economyMux.HandleFunc(pat.Post("/shop/:id/delete"), deleteItemHandler)
	economyMux.HandleFunc(pat.Post("/shop/:id/delete/"), deleteItemHandler)
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
	err = decoder.Decode(responseEntry, r.PostForm)
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

type ItemField string

const (
	DescriptionField ItemField = "Description"
	PriceField       ItemField = "Price"
	QuantityField    ItemField = "Quantity"
	ReplyField       ItemField = "Reply"
)

func saveNewItemHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	guildID := pat.Param(r, "server")
	guild := functions.GetGuild(guildID)

	var item models.EconomyShop

	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err := decoder.Decode(&item, r.PostForm)
	if err != nil {
		web.SendErrorToast(w, fmt.Sprintf("Failed to decode form: %s", err.Error()))
		return
	}

	item.GuildID = guild.ID

	ok, err := isItemOk(&item, false)
	if !ok {
		web.SendErrorToast(w, err.Error())
		return
	}

	err = item.Insert(context.Background(), common.PQ, boil.Infer())
	if err != nil {
		web.SendErrorToast(w, fmt.Sprintf("Failed to save item: %s", err.Error()))
		return
	}

	web.SendSuccessToast(w, "Successfully saved item.")
}

func editItemHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	guildID := pat.Param(r, "server")
	guild := functions.GetGuild(guildID)

	itemID := pat.Param(r, "id")
	itemIDInt, _ := strconv.Atoi(itemID)

	itemEntry, err := models.EconomyShops(models.EconomyShopWhere.GuildID.EQ(guild.ID), models.EconomyShopWhere.ID.EQ(itemIDInt)).One(context.Background(), common.PQ)
	if err != nil {
		web.SendErrorToast(w, "Item not found.")
		return
	}

	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err = decoder.Decode(itemEntry, r.PostForm)
	if err != nil {
		web.SendErrorToast(w, fmt.Sprintf("Failed to decode form: %s", err.Error()))
		return
	}

	itemEntry.GuildID = guild.ID

	ok, err := isItemOk(itemEntry, true)
	if !ok {
		web.SendErrorToast(w, err.Error())
		return
	}

	err = itemEntry.Insert(context.Background(), common.PQ, boil.Infer())
	if err != nil {
		web.SendErrorToast(w, fmt.Sprintf("Failed to save item: %s", err.Error()))
		return
	}

	web.SendSuccessToast(w, "Successfully saved item.")
}

func deleteItemHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	guildID := pat.Param(r, "server")
	guild := functions.GetGuild(guildID)

	itemID := pat.Param(r, "id")
	itemIDInt, _ := strconv.Atoi(itemID)

	itemEntry, err := models.EconomyShops(models.EconomyShopWhere.GuildID.EQ(guild.ID), models.EconomyShopWhere.ID.EQ(itemIDInt)).One(context.Background(), common.PQ)
	if err != nil {
		web.SendErrorToast(w, "Item not found.")
		return
	}

	itemEntry.Delete(context.Background(), common.PQ)
	web.SendSuccessToast(w, "Successfully deleted.")
}

func isItemOk(item *models.EconomyShop, ignoreCurrentName bool) (bool, error) {
	ok, err := itemNameOk(item.GuildID, item.Name, ignoreCurrentName)
	if !ok {
		return ok, err
	}

	ok, err = itemDescReplyOkay(DescriptionField, item.Description)
	if !ok {
		return ok, err
	}

	ok, err = itemDescReplyOkay(ReplyField, item.Reply)
	if !ok {
		return ok, err
	}

	ok, err = itemPriceQuantityOkay(PriceField, item.Price)
	if !ok {
		return ok, err
	}

	ok, err = itemPriceQuantityOkay(QuantityField, item.Quantity)
	if !ok {
		return ok, err
	}

	ok, err = itemRoleOkay(item.GuildID, item.Role)
	if !ok {
		return ok, err
	}

	return true, nil
}

func itemNameOk(guildID string, newName string, ignoreCurrentName bool) (bool, error) {
	if newName == "" {
		return false, errors.New("Name cannot be empty.")
	}

	if !ignoreCurrentName {
		currentItem, _ := models.EconomyShops(models.EconomyShopWhere.GuildID.EQ(guildID), models.EconomyShopWhere.Name.EQ(newName)).One(context.Background(), common.PQ)
		if currentItem != nil {
			return false, errors.New("Item with this name already exists.")
		}
	}

	maxLength := 30
	if len(newName) > maxLength {
		return false, fmt.Errorf("The name must be less than %d characters.", maxLength)
	}

	return true, nil
}

func itemDescReplyOkay(t ItemField, s string) (bool, error) {
	maxLength := 200
	if len(s) > maxLength {
		return false, fmt.Errorf("The %s must be less than %d characters", t, maxLength)
	}

	return true, nil
}

func itemPriceQuantityOkay(t ItemField, n int64) (bool, error) {
	if n < 0 {
		return false, fmt.Errorf("The %s must be a positive number.", t)
	}

	return true, nil
}

func itemRoleOkay(guildID string, roleID string) (bool, error) {
	if roleID == "" {
		return true, nil
	}

	role, err := functions.GetRole(guildID, roleID)
	if err != nil || role == nil {
		return false, errors.New("Role not found.")
	}

	return true, nil
}
