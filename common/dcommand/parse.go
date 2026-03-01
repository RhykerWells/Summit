package dcommand

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/durationutil"
	"github.com/bwmarrin/discordgo"
)

// ParsedArg represents a single argument parsed from a command invocation.
// Each argument contains its name, expected type, and the supplied value.
type ParsedArg struct {
	Name  string
	Type  ArgumentType
	Value interface{} // raw value provided by the user
}

// String returns the argument's string representation.
// It safely converts the underlying value to a string, supporting primitive numeric types.
func (p *ParsedArg) String() string {
	if p.Value == nil {
		return ""
	}

	switch t := p.Value.(type) {
	case string:
		return t
	case int, int32, int64, uint, uint32, uint64:
		return strconv.FormatInt(functions.ToInt64(t), 10)
	default:
		return ""
	}
}

// Int64 converts and returns the argument's value as an int64.
// If the value is nil or non-numeric, it safely returns 0.
func (p *ParsedArg) Int64() int64 {
	if p.Value == nil {
		return 0
	}

	return functions.ToInt64(p.Value)
}

// User attempts to resolve the argument into a Discord user.
// If the value is nil or resolution fails, it returns nil.
func (p *ParsedArg) User() *discordgo.User {
	if p.Value == nil {
		return nil
	}

	user, _ := functions.GetUser(p.String())
	return user
}

// Member attempts to resolve the argument into a Discord guild member.
// If the value is nil or resolution fails, it returns nil.
func (p *ParsedArg) Member(guildID string) *discordgo.Member {
	if p.Value == nil {
		return nil
	}

	member, _ := functions.GetMember(guildID, p.String())
	return member
}

// BetAmount returns the argument's lowercase string value, trimmed of any surrounding whitespace.
// Typically used for betting-related arguments (e.g., "all", "half", or a specific amount).
func (p *ParsedArg) BetAmount() string {
	if p.Value == nil {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(p.String()))
}

// Duration attempts to parse the argument into a time.Duration pointer.
// Returns nil if parsing fails or if the argument has no value.
func (p *ParsedArg) Duration() *time.Duration {
	if p.Value == nil {
		return nil
	}

	duration, _ := durationutil.ToDuration(p.String())
	return &duration
}

// Coin returns the argument's lowercase string value, trimmed of whitespace.
// Used for commands that require a coin flip guess, e.g., "heads" or "tails".
func (p *ParsedArg) Coin() string {
	if p.Value == nil {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(p.String()))
}

// BalanceType returns the argument's lowercase string value, trimmed of whitespace.
// Used for commands that expect a balance source argument such as "bank" or "cash".
func (p *ParsedArg) BalanceType() string {
	if p.Value == nil {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(p.String()))
}

// handleMissingOrInvalidArgs
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
	display := ""
	for _, combo := range cmd.ArgumentCombos {
		display += "\n" + cmd.Command
		for i, idx := range combo {
			if idx < 0 || idx >= len(cmd.Args) {
				continue
			}

			arg := cmd.Args[idx]
			if i < cmd.ArgsRequired {
				display += " <" + arg.Name + ":" + arg.Type.Help() + ">"
			} else {
				display += " [" + arg.Name + ":" + arg.Type.Help() + "]"
			}
		}
	}

	return errors.New("Invalid argument order or types"), errorEmbed(cmd.Command, data, fmt.Sprintf("Invalid argument order or types\nExpected one of:\n```%s %s```", cmd.Command, display))
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
