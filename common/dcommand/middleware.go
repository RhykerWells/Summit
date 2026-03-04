package dcommand

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/RhykerWells/Summit/common"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

// hasRequiredPermissions runs the permission validation for the user & bot in the current channel
func hasRequiredPermissions(cmd SummitCommand, data *Data) error {
	logrus.Infoln("In permission checker")
	err := hasUserPermissions(cmd, data)
	if err != nil {
		return err
	}

	err = hasBotPermissions(cmd, data)
	if err != nil {
		return err
	}

	return nil
}

// hasUserPermissions validates that the user has one of any permissions required
func hasUserPermissions(cmd SummitCommand, data *Data) error {
	if len(cmd.RequiredUserPerms) < 1 {
		return nil
	}

	perms, _ := common.Session.State.UserChannelPermissions(data.Author.ID, data.ChannelID)

	permsMet := false
	for _, perm := range cmd.RequiredUserPerms {
		if perms&perm != 0 {
			permsMet = true
			break
		}
	}

	if !permsMet {
		humanisedPerms := humanisePerms(cmd.RequiredUserPerms)
		return fmt.Errorf("You do not have any of the required permissions to run this command.\nThis command required one of the following permissions: %s", humanisedPerms)
	}

	return nil
}

// hasBotPermissions validates that the bot has one of any permissions required
func hasBotPermissions(cmd SummitCommand, data *Data) error {
	if len(cmd.RequiredBotPerms) < 1 {
		return nil
	}

	perms, _ := common.Session.State.UserChannelPermissions(common.Bot.ID, data.ChannelID)

	permsMet := false
	for _, perm := range cmd.RequiredUserPerms {
		if perms&perm != 0 {
			permsMet = true
			break
		}
	}

	if !permsMet {
		humanisedPerms := humanisePerms(cmd.RequiredBotPerms)
		return fmt.Errorf("The bot does not have any of the required permissions to run this command.\nThis command required one of the following permissions: %s", humanisedPerms)
	}

	logrus.Infoln(humanisePerms(cmd.RequiredUserPerms))

	return nil
}

func humanisePerms(perms []int64) string {
	humanisedPerms := make([]string, 0, len(perms))
	for _, perm := range perms {
		humanisedPerms = append(humanisedPerms, fmt.Sprintf("`%s`", permissionName(perm)))
	}

	return strings.Join(humanisedPerms, " or ")
}

// handleMissingOrInvalidArgs validates any required arguments/argument combinations
func handleMissingOrInvalidArgs(cmd SummitCommand, data *Data, tokens []string) (error, *discordgo.MessageEmbed) {
	// If no combos defined, use positional parsing
	if len(cmd.ArgumentCombos) == 0 {

		// Build display string
		display := ""
		for i, a := range cmd.Args {
			if i < cmd.ArgsRequired {
				display += " <" + a.Name + ":" + a.Type.Help() + ">"
			} else {
				display += " [" + a.Name + ":" + a.Type.Help() + "]"
			}
		}

		// ensure we satisfy the required arg count
		if len(tokens) < cmd.ArgsRequired {
			return errors.New("Token number under threshhold"), errorEmbed(cmd.Command, data, fmt.Sprintf("Missing required arguments\n```%s %s```", cmd.Command, display))
		}

		parsedArgs := []*ParsedArg{}
		for i, arg := range cmd.Args {
			var value string
			if i < len(tokens) {
				value = tokens[i]
				// Last string absorbs remaining tokens
				if i == len(cmd.Args)-1 && arg.Type == String && len(tokens) > i {
					value = strings.Join(tokens[i:], " ")
				}
			} else {
				parsedArgs = append(parsedArgs, &ParsedArg{Name: arg.Name, Type: arg.Type, Value: nil})
				continue
			}

			parsedArgs = append(parsedArgs, &ParsedArg{
				Name:  arg.Name,
				Type:  arg.Type,
				Value: value,
			})
		}

		// Validate only supplied arguments
		for _, pArg := range parsedArgs {
			if pArg.Value == nil {
				continue
			}
			if !pArg.Type.ValidateArg(pArg, data) {
				return errors.New("arg value does not match required type"), errorEmbed(cmd.Command, data, fmt.Sprintf("Invalid `%s` argument. Expected: `%s`", pArg.Name, pArg.Type.Help()))
			}
		}

		data.ParsedArgs = parsedArgs
		return nil, nil
	}

	// --- Handle combos  ---
	for _, combo := range cmd.ArgumentCombos {

		// If there are fewer tokens than combo entries, this combo can't match
		if len(tokens) < len(combo) {
			continue
		}

		// Determine if we have extra tokens and, if so, which argument can absorb them.
		absorbIdx := -1
		if len(tokens) > len(combo) {
			for i := len(combo) - 1; i >= 0; i-- {
				if cmd.Args[combo[i]].Type == String {
					absorbIdx = i
					break
				}
			}
			if absorbIdx == -1 {
				// no string argument available to take the overflow
				continue
			}
		}

		// Distribute the tokens to the arguments, with the last string argument absorbing any leftover words
		ok := true
		tokenIdx := 0

		for i, defPos := range combo {
			def := cmd.Args[defPos]
			var token string

			if absorbIdx != -1 && i == absorbIdx {
				// This argument swallows either all remaining tokens (if it's last)
				// or all tokens up until there are enough left for the subsequent
				// arguments.
				if absorbIdx == len(combo)-1 {
					token = strings.Join(tokens[tokenIdx:], " ")
					tokenIdx = len(tokens)
				} else {
					rems := len(combo) - (i + 1) // number of args after this one
					end := len(tokens) - rems
					if end <= tokenIdx {
						ok = false
						break
					}
					token = strings.Join(tokens[tokenIdx:end], " ")
					tokenIdx = end
				}
			} else {
				if tokenIdx >= len(tokens) {
					ok = false
					break
				}
				token = tokens[tokenIdx]
				tokenIdx++
			}

			if !def.Type.ValidateArg(&ParsedArg{Type: def.Type, Value: token}, data) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}

		// Build parsed args according to the combo mapping using similar logic
		parsedArgs := make([]*ParsedArg, len(cmd.Args))
		tokenIdx = 0
		for i, defPos := range combo {
			def := cmd.Args[defPos]
			var val string

			if absorbIdx != -1 && i == absorbIdx {
				if absorbIdx == len(combo)-1 {
					val = strings.Join(tokens[tokenIdx:], " ")
					tokenIdx = len(tokens)
				} else {
					rems := len(combo) - (i + 1)
					end := len(tokens) - rems
					val = strings.Join(tokens[tokenIdx:end], " ")
					tokenIdx = end
				}
			} else {
				val = tokens[tokenIdx]
				tokenIdx++
			}

			parsedArgs[defPos] = &ParsedArg{
				Name:  def.Name,
				Type:  def.Type,
				Value: val,
			}
		}

		data.ParsedArgs = parsedArgs
		return nil, nil
	}

	// No combo matched
	parts := []string{}
	for _, combo := range cmd.ArgumentCombos {
		var s strings.Builder
		s.WriteString(cmd.Command)
		for _, idx := range combo {
			if idx < 0 || idx >= len(cmd.Args) {
				continue
			}
			argHelp := cmd.Args[idx].Name + ":" + cmd.Args[idx].Type.Help()
			s.WriteString(" <" + argHelp + ">")
		}
		parts = append(parts, s.String())
	}
	display := strings.Join(parts, "\n")

	return errors.New("Invalid arguments"), errorEmbed(cmd.Command, data, fmt.Sprintf("%s", "No matching argument combination found.\nExpected one of:\n```\n"+display+"\n```"))
}

// errorEmbed constructs and returns a standardized error embed for a command execution failure.
// It includes the author's username, avatar, timestamp, and an error description.
func errorEmbed(cmd string, data *Data, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    data.Author.Username + " - " + cmd,
			IconURL: data.Author.AvatarURL("256"),
		},
		Timestamp:   time.Now().Format(time.RFC3339),
		Color:       common.ErrorRed,
		Description: description,
	}
}
