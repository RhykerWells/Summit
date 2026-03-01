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
	Name     string
	Type     ArgumentType
}

type ArgumentType interface {
	ValidateArg(arg *ParsedArg, data *Data) bool
	Help() string
}

var (
	String      = &StringArg{}
	Int         = &IntArg{}
	User        = &UserArg{}
	Member      = &MemberArg{}
	Bet         = &BetArg{}
	Duration    = &DurationArg{}
	Coin        = &CoinArg{}
	UserBalance = &BalanceArg{}
)

var (
	_ ArgumentType = (*StringArg)(nil)
	_ ArgumentType = (*IntArg)(nil)
	_ ArgumentType = (*UserArg)(nil)
	_ ArgumentType = (*MemberArg)(nil)
	_ ArgumentType = (*BetArg)(nil)
	_ ArgumentType = (*DurationArg)(nil)
	_ ArgumentType = (*CoinArg)(nil)
	_ ArgumentType = (*BalanceArg)(nil)
)

type StringArg struct{}

func (s *StringArg) Help() string {
	return "Text"
}

func (s *StringArg) ValidateArg(arg *ParsedArg, data *Data) bool {
	return true
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
	var maxStr string
	if b.Max != 0 {
		maxStr = fmt.Sprintf(" and below %d", b.Max)
	}
	return fmt.Sprintf("Whole number above %d%s|Max|All", b.Min, maxStr)
}

func (b *BetArg) ValidateArg(arg *ParsedArg, data *Data) bool {
	vStr := strings.ToLower(strings.TrimSpace(arg.Value.(string)))

	// Allow keywords
	if vStr == "max" || vStr == "all" {
		return true
	}

	// Validate integer via regex or conversion
	matched, _ := regexp.MatchString(`^-?\d+$`, vStr)
	if !matched {
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

type CoinArg struct{}

func (c *CoinArg) Help() string {
	return "Heads/Tails"
}

func (c *CoinArg) ValidateArg(arg *ParsedArg, data *Data) bool {
	vStr := strings.ToLower(strings.TrimSpace(arg.Value.(string)))

	// Allow keywords
	if vStr != "heads" && vStr != "tails" {
		return false
	}

	return true
}

type BalanceArg struct{}

func (b *BalanceArg) Help() string {
	return "Bank/Cash"
}

func (b *BalanceArg) ValidateArg(arg *ParsedArg, data *Data) bool {
	vStr := strings.ToLower(strings.TrimSpace(arg.Value.(string)))

	// Allow keywords
	if vStr != "bank" && vStr != "cash" {
		return false
	}

	return true
}
