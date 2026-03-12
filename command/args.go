package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/RhykerWells/dispatch"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var _ dispatch.ArgumentType = (*BetArg)(nil)

type BetArg struct {
	Options []string
	Min     int64
	Max     int64
}

func (b *BetArg) Help() string {
	var rangeDesc string
	if b.Max != 0 {
		rangeDesc = fmt.Sprintf("%d-%d", b.Min, b.Max)
	} else {
		rangeDesc = fmt.Sprintf("%d+", b.Min)
	}

	var options string
	for _, option := range b.Options {
		options += fmt.Sprintf("/%s", cases.Title(language.Und).String(option))
	}

	return fmt.Sprintf("%s%s", rangeDesc, options)
}

func (b *BetArg) ValidateArg(arg *dispatch.ParsedArg, data *dispatch.Data) (any, bool) {
	vStr := strings.TrimSpace(arg.Raw)

	// Allow keywords
	for _, option := range b.Options {
		if strings.EqualFold(vStr, option) {
			return strings.ToLower(vStr), true
		}
	}

	// Validate integer via regex or conversion
	vInt, err := strconv.ParseInt(vStr, 0, 64)
	if err != nil {
		return nil, false
	}

	if vInt < b.Min {
		return nil, false
	}

	if b.Max != 0 && vInt > b.Max {
		return nil, false
	}

	return vInt, true
}
