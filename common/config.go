package common

import (
	"os"
	"strings"
)

var (
	ConfigBotToken    = os.Getenv("SUMMIT_TOKEN")
	ConfigBotClientID = os.Getenv("SUMMIT_CLIENTID")
	ConfigBotSecret   = os.Getenv("SUMMIT_CLIENTSECRET")

	ConfigPGHost     = os.Getenv("SUMMIT_PGHOST")
	ConfigPGPort     = os.Getenv("SUMMIT_PGPORT")
	ConfigPGDB       = os.Getenv("SUMMIT_PGDB")
	ConfigPGUsername = os.Getenv("SUMMIT_PGUSER")
	ConfigPGPassword = os.Getenv("SUMMIT_PGPASSWORD")

	ConfigSummitHost         = os.Getenv("SUMMIT_HOST")
	ConfigTermsURLOverride   = os.Getenv("SUMMIT_TERMSURLOVERRIDE")
	ConfigPrivacyURLOverride = os.Getenv("SUMMIT_PRIVACYURLOVERRIDE")

	ConfigBotOwner  = os.Getenv("SUMMIT_OWNERID")
	ConfigSupportID = os.Getenv("SUMMIT_SERVERID")

	ConfigGitHubRepo      = os.Getenv("SUMMIT_GITHUBREPO")
	ConfigGitHubRepoOwner = os.Getenv("SUMMIT_GITHUBREPOOWNER")
)

// ConfigDgoBotToken prefixes the bot token with the required "Bot " if it is not done by the host
func ConfigDgoBotToken() string {
	token := ConfigBotToken
	if !strings.HasPrefix(token, "Bot ") {
		token = "Bot " + token
	}
	return token
}
