package dcommand

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

var (
	CategoryGeneral = CommandCategory{
		Name:        "General",
		Description: "General bot commands",
	}
	CategoryOwner = CommandCategory{
		Name:        "Owner",
		Description: "Mainanance and other bot-owner commands",
	}
	CategoryEconomy = CommandCategory{
		Name:        "Economy",
		Description: "Gambling and other economy based commands",
	}
	CategoryModeration = CommandCategory{
		Name:        "Moderation",
		Description: "Moderation and guild safety",
	}
)

// SummitCommand defines the general data that must be set during the addition of a new command
type SummitCommand struct {
	Command     string
	Category    CommandCategory
	Aliases     []string
	Description string

	Args           []*Arg
	ArgsRequired   int // Ignored if using combos
	ArgumentCombos [][]int

	Run Run
}

// CommandCategory defines the available category types for commands
type CommandCategory struct {
	Name        string
	Description string
}

// CommandHandler defines the general command handler, the full instances of a command and a string map to retireve them
type CommandHandler struct {
	cmdInstances []SummitCommand
	cmdMap       map[string]SummitCommand
}

// RegisteredCommand defines the context required to access data surrounding a command
type RegisteredCommand struct {
	Trigger        string
	Category       CommandCategory
	Aliases        []string
	Description    string
	Args           []*Arg
	RequiredArgs   int
	ArgumentCombos [][]int
}

// RegisterCommands adds each command to the command handler
func (c *CommandHandler) RegisterCommands(cmds ...*SummitCommand) {
	for _, cmd := range cmds {
		c.cmdInstances = append(c.cmdInstances, *cmd)

		// Limit aliases to 3 max
		if len(cmd.Aliases) > 3 {
			aliasOver := len(cmd.Aliases) - 3
			cmd.Aliases = cmd.Aliases[:3]
			logrus.Warnln(fmt.Sprintf("%s has %[2]d too many aliases. Automatically removed the last %[2]d.", cmd.Command, aliasOver))
		}

		// Register main command
		c.cmdMap[cmd.Command] = *cmd

		// Register aliases
		for _, alias := range cmd.Aliases {
			c.cmdMap[alias] = *cmd
		}
	}
}

// RegisteredCommands returns an array of each RegisteredCommand
func (c *CommandHandler) RegisteredCommands() map[string]RegisteredCommand {
	cmdMap := make(map[string]RegisteredCommand)
	for _, cmd := range c.cmdMap {
		rcmd := &RegisteredCommand{
			Trigger:        cmd.Command,
			Category:       cmd.Category,
			Aliases:        cmd.Aliases,
			Description:    cmd.Description,
			Args:           cmd.Args,
			RequiredArgs:   cmd.ArgsRequired,
			ArgumentCombos: cmd.ArgumentCombos,
		}
		cmdMap[cmd.Command] = *rcmd

		// Also register aliases as keys for lookup
		for _, alias := range cmd.Aliases {
			cmdMap[alias] = *rcmd
		}
	}
	return cmdMap
}

type Run func(data *Data)
