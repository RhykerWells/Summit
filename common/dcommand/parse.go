package dcommand

import (
	"strconv"
	"strings"
	"time"

	"github.com/RhykerWells/Summit/bot/functions"
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
