package dcommand

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/durationutil"
)

// Arg defines the structure to pass argument data with
type Arg struct {
	Name string
	Type ArgumentType
}

type ArgumentType interface {
	ValidateArg(arg *ParsedArg, data *Data) bool
	Help() string
}

var (
	String   = &StringArg{}
	Int      = &IntArg{}
	User     = &UserArg{}
	Member   = &MemberArg{}
	Bet      = &BetArg{}
	Duration = &DurationArg{}
)

var (
	_ ArgumentType = (*StringArg)(nil)
	_ ArgumentType = (*IntArg)(nil)
	_ ArgumentType = (*UserArg)(nil)
	_ ArgumentType = (*MemberArg)(nil)
	_ ArgumentType = (*BetArg)(nil)
	_ ArgumentType = (*DurationArg)(nil)
)

type StringArg struct {
	Options []string
}

func (s *StringArg) Help() string {
	if len(s.Options) > 0 {
		return fmt.Sprintf("%s", strings.Join(s.Options, "/"))
	}

	return "Text"
}

func (s *StringArg) ValidateArg(arg *ParsedArg, data *Data) bool {
	v := arg.Value.(string)
	if len(s.Options) > 0 {
		for _, option := range s.Options {
			if strings.EqualFold(v, option) {
				return true
			}
		}
		return false
	}

	return v != ""
}

type IntArg struct {
	Min int64
	Max int64
}

func (i *IntArg) Help() string {
	var maxStr string
	if i.Max != 0 {
		maxStr = fmt.Sprintf(" and below %d", i.Max)
	}
	return fmt.Sprintf("Whole number above %d%s", i.Min, maxStr)
}

func (i *IntArg) ValidateArg(arg *ParsedArg, data *Data) bool {
	v := functions.ToInt64(arg.Value)
	if v < i.Min {
		return false
	}
	if i.Max != 0 && v > i.Max {
		return false
	}

	return true
}

type UserArg struct{}

func (u *UserArg) Help() string {
	return "Mention/ID"
}

func (u *UserArg) ValidateArg(arg *ParsedArg, data *Data) bool {
	v := arg.Value.(string)
	_, err := functions.GetUser(v)

	return err == nil
}

type MemberArg struct{}

func (m *MemberArg) Help() string {
	return "Mention/ID"
}

func (m *MemberArg) ValidateArg(arg *ParsedArg, data *Data) bool {
	v := arg.Value.(string)
	_, err := functions.GetMember(data.GuildID, v)

	return err == nil
}

type BetArg struct {
	Min int64
	Max int64
}

func (b *BetArg) Help() string {
	var rangeDesc string
	if b.Max != 0 {
		rangeDesc = fmt.Sprintf("%d-%d", b.Min, b.Max)
	} else {
		rangeDesc = fmt.Sprintf("%d+", b.Min)
	}
	return fmt.Sprintf("%s/Max/All", rangeDesc)
}

var intRegex = regexp.MustCompile(`^-?\d+$`)

func (b *BetArg) ValidateArg(arg *ParsedArg, data *Data) bool {
	vStr := strings.ToLower(strings.TrimSpace(arg.Value.(string)))

	// Allow keywords
	if vStr == "max" || vStr == "all" {
		return true
	}

	// Validate integer via regex or conversion
	if !intRegex.MatchString(vStr) {
		return false
	}

	v := functions.ToInt64(vStr)
	if v < b.Min {
		return false
	}
	if b.Max != 0 && v > b.Max {
		return false
	}

	return true
}

type DurationArg struct{}

func (d *DurationArg) Help() string {
	return "Duration"
}

func (d *DurationArg) ValidateArg(arg *ParsedArg, data *Data) bool {
	v := arg.Value.(string)
	_, err := durationutil.ToDuration(v)

	return err == nil
}
