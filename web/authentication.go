package web

import (
	"net/http"

	"github.com/RhykerWells/Summit/common"
	"golang.org/x/oauth2"
)

var OauthConf *oauth2.Config

// initDiscordOauthHandles the creating Oauth2 config for authenticating user sign ins
func initDiscordOauth() {
	OauthConf = &oauth2.Config{
		ClientID:     common.ConfigBotClientID,
		ClientSecret: common.ConfigBotSecret,
		Scopes:       []string{"identify", "guilds"},
		Endpoint: oauth2.Endpoint{
			TokenURL: "https://discord.com/api/oauth2/token",
			AuthURL:  "https://discord.com/api/oauth2/authorize",
		},
	}
	OauthConf.RedirectURL = "https://" + common.ConfigSummitHost + "/confirm"
}

// handleLogin attempts to log in a user if they have a valid session, otherwise redirects them to the specified Auth URL
func handleLogin(w http.ResponseWriter, r *http.Request) {
	// checks for valid user session and automatically redirect if exists
	_, err := checkUserCookie(w, r)
	if err == nil {
		http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
		return
	}

	// generates a CSRF token and begins the login sequence
	csrfToken, err := createCSRF()
	if err != nil {
		return
	}
	setCSRF(w, csrfToken)
	url := OauthConf.AuthCodeURL(csrfToken, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// confirmLogin handles the successful Discord Oauth login and redirects users to the dashboard
func confirmLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	csrf := getCSRF(w, r)
	state := r.URL.Query().Get("state")
	if state != csrf {
		http.Redirect(w, r, "/?error=invalid_CSRF", http.StatusSeeOther)
		return
	}

	code := r.FormValue("code")
	token, err := OauthConf.Exchange(ctx, code)
	if err != nil {
		http.Redirect(w, r, "/?error=oauth2_failure", http.StatusSeeOther)
		return
	}

	client := OauthConf.Client(ctx, token)
	resp, err := client.Get("https://discord.com/api/v10/users/@me")
	if err != nil {
		http.Redirect(w, r, "/?error=failed_retrieving_info", http.StatusSeeOther)
		return
	}
	defer resp.Body.Close()

	setUserSession(w, token)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// handleLogout handles the logout route and ensures that all cookies related to data storage are removed
func handleLogout(w http.ResponseWriter, r *http.Request) {
	if userCookie, err := r.Cookie("summit_userinfo"); err == nil {
		deleteCookie(w, userCookie)
	}

	if csrfCookie, err := r.Cookie("summit_csrf"); err == nil {
		deleteCookie(w, csrfCookie)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
