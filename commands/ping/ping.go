package ping

import (
	"time"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/common/dcommand"
)

var Command = &dcommand.SummitCommand{
	Command:     "ping",
	Category:    dcommand.CategoryGeneral,
	Description: "Displays bot latency",
	Run: (func(data *dcommand.Data) error {
		msg, err := functions.SendBasicMessage(data.ChannelID, "Ping...")
		if err != nil {
			return nil
		}

		started := time.Now()
		functions.EditBasicMessage(msg.ChannelID, msg.ID, "Pong! (Edit): "+(time.Since(started)*time.Microsecond).String())

		return nil
	}),
}
