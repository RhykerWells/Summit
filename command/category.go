package command

import "github.com/RhykerWells/dispatch"

var (
	CategoryGeneral = dispatch.CommandCategory{
		Name:        "General",
		Description: "General bot commands",
	}
	CategoryOwner = dispatch.CommandCategory{
		Name:        "Owner",
		Description: "Mainanance and other bot-owner commands",
	}
	CategoryEconomy = dispatch.CommandCategory{
		Name:        "Economy",
		Description: "Gambling and other economy based commands",
	}
	CategoryModeration = dispatch.CommandCategory{
		Name:        "Moderation",
		Description: "Moderation and guild safety",
	}
	CategoryMisc = dispatch.CommandCategory{
		Name:        "Misc",
		Description: "Commands that don't fit into other categories",
	}
)
