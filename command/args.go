package command

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/dispatch"
)

var _ dispatch.ArgumentType = (*BetArg)(nil)

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

func (b *BetArg) ValidateArg(arg *dispatch.ParsedArg, data *dispatch.Data) (any, bool) {
	vStr := strings.ToLower(strings.TrimSpace(arg.Value.(string)))

	// Allow keywords
	if vStr == "max" || vStr == "all" {
		return vStr, true
	}

	// Validate integer via regex or conversion
	if !intRegex.MatchString(vStr) {
		return nil, false
	}

	vInt := functions.ToInt64(vStr)
	if vInt < b.Min {
		return nil, false
	}
	if b.Max != 0 && vInt > b.Max {
		return nil, false
	}

	return vInt, true
}
