package ping

import (
	"time"

	"github.com/RhykerWells/Summit/bot/functions"
	"github.com/RhykerWells/Summit/command"
	"github.com/RhykerWells/dispatch"
)

var Command = &dispatch.Command{
	Command:     "ping",
	Category:    command.CategoryGeneral,
	Description: "Displays bot latency",
	Run: (func(data *dispatch.Data) error {
		msg, err := functions.SendBasicMessage(data.Channel.ID, "Ping...")
		if err != nil {
			return nil
		}

		started := time.Now()
		functions.EditBasicMessage(msg.ChannelID, msg.ID, "Pong! (Edit): "+(time.Since(started)*time.Microsecond).String())

		return nil
	}),
}
