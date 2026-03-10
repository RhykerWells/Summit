package help

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/dispatch"
	"github.com/bwmarrin/discordgo"
)

var Command = &dispatch.Command{
	Command:  "help",
	Aliases:  []string{"h"},
	Category: command.CategoryGeneral,
	Args: []*dispatch.Arg{
		{Name: "Command", Type: dispatch.String},
	},
	Description: "Displays bot help",
	Run: func(data *dispatch.Data) error {
		command := ""
		if len(data.ParsedArgs) > 0 {
			command = data.ParsedArgs[0].Value.(string)
		}

		// Per-command help
		if command != "" {
			help(command, data.Channel.ID)
			return nil
		}

		// Generic help category
		genericCategoryHelp(data.Channel.ID)

		return nil
	},
}

// genericCategoryHelp builds and sends an embed listing all available categories
// and their commands. The "General" category is always listed first, followed
// by the other categories sorted alphabetically.
func genericCategoryHelp(channelID string) {
	cmdMap := command.CommandHandler.RegisteredCommands()
	categories := make(map[string][]string)
	for _, cmd := range cmdMap {
		categories[cmd.Command.Category.Name] = append(categories[cmd.Command.Category.Name], cmd.Command.Command)
	}
	categoryNames := make([]string, 0, len(categories))
	for categoryName := range categories {
		categoryNames = append(categoryNames, categoryName)
	}
	sort.SliceStable(categoryNames, func(i, j int) bool {
		if categoryNames[i] == "General" {
			return true
		}
		if categoryNames[j] == "General" {
			return false
		}
		return categoryNames[i] < categoryNames[j]
	})

	helpEmbed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    fmt.Sprintf("%s help", common.Bot.Username),
			IconURL: common.Bot.AvatarURL("256"),
		},
		Description: "Here are the available categories and commands:",
		Color:       common.SuccessGreen,
	}
	for _, categoryName := range categoryNames {
		// Sort commands within the category
		sort.Strings(categories[categoryName])
		categoryStr := fmt.Sprintf("**%s**: `%s`", categoryName, strings.Join(categories[categoryName], "`, `"))
		helpEmbed.Description += "\n\n" + categoryStr
	}

	message := &discordgo.MessageSend{
		Embed: helpEmbed,
	}
	functions.SendMessage(channelID, message)
}

// help shows detailed help for a specific command, including its description,
// aliases, and expected arguments. If the command cannot be found, a simple
// error message is sent instead.
func help(commandName string, channelID string) {
	cmdMap := command.CommandHandler.RegisteredCommands()
	cmd, ok := cmdMap[commandName]
	if !ok {
		functions.SendBasicMessage(channelID, fmt.Sprintf("Command `%s` not found", commandName))
		return
	}
	aliases := ""
	if len(cmd.Command.Aliases) >= 1 {
		aliases = fmt.Sprintf("/%s", strings.Join(cmd.Command.Aliases, "/"))
	}
	helpEmbed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    fmt.Sprintf("%s help - %s%s", common.Bot.Username, commandName, aliases),
			IconURL: common.Bot.AvatarURL("256"),
		},
		Description: cmd.Command.Description,
		Color:       common.SuccessGreen,
	}
	args := getArgs(cmd)
	helpEmbed.Description = cmd.Command.Description
	if args != "" {
		helpEmbed.Description += "\n```" + args + "\n```"
	}
	message := &discordgo.MessageSend{
		Embed: helpEmbed,
	}
	functions.SendMessage(channelID, message)
}

// getArgs builds the formatted string of arguments for a given command.
// Required arguments are enclosed in <angle brackets>, and optional arguments
// are enclosed in [square brackets].
func getArgs(command dispatch.RegisteredCommand) (str string) {
	// If explicit argument combos are provided, show them as alternatives
	if len(command.Command.ArgumentCombos) > 0 {
		parts := []string{}
		for _, combo := range command.Command.ArgumentCombos {
			var s strings.Builder
			s.WriteString(command.Command.Command) // Trigger once at start
			for _, idx := range combo {
				if idx < 0 || idx >= len(command.Command.Args) {
					continue
				}
				s.WriteString(" <" + argHelp(command.Command.Args[idx]) + ">") // Space before arg
			}
			parts = append(parts, s.String())
		}

		return strings.Join(parts, "\n")
	}

	// Fallback: use RequiredArgs to mark which are required vs optional
	str = command.Command.Command
	for i, arg := range command.Command.Args {
		if i < command.Command.ArgsRequired {
			str += " <" + argHelp(arg) + ">"
		} else {
			str += " [" + argHelp(arg) + "]"
		}
	}
	return str
}

// argHelp returns a formatted string for a single argument, showing both its
// name and type.
func argHelp(arg *dispatch.Arg) (str string) {
	argType := arg.Type.Help()
	str = fmt.Sprintf("%s:%s", arg.Name, argType)
	return
}
