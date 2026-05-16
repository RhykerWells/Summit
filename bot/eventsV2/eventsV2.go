package eventsv2

//go:generate go run gen/eventsV2_gen.go

import (
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

type EventData struct {
	Interface interface{}
	Type      Event
	Session   *discordgo.Session
}

type HandlerFunc func(*EventData) error

// We'll keep handlers in their own struct in case we want to add more functionality to them in the future
type Handler struct {
	F HandlerFunc
}

var handlers = make([][]*Handler, 98)

// AddHandler adds a event handler
func AddHandler(handler HandlerFunc, evts ...Event) {
	h := &Handler{
		F: handler,
	}

	for _, evt := range evts {
		handlers[evt] = append(handlers[evt], h)
	}
}

// This is the one event handler that the session needs to attach, it will register handlers for all events, for the bot to use them when necessary
func HandleEvent(session *discordgo.Session, event interface{}) {
	var evt = &EventData{
		Interface: event,
		Session:   session,
	}

	setEventType(evt)

	// Maybe add internal events here in the future?
	// How could we add user defined events? Maybe we could have a function that registers user defined events, and then we could check if the event is a user defined event before checking for internal events?

	EmitEvent(evt)
}

// EmitEvent executes all registered handlers for the event type,
// copying them to avoid concurrent modifications during execution.
func EmitEvent(data *EventData) {
	h := make([]*Handler, len(handlers[data.Type]))
	copy(h, handlers[data.Type])
	runEvents(h, data)
}

func runEvents(h []*Handler, data *EventData) {
	for _, v := range h {
		func() {
			defer func() {
				if errI := recover(); errI != nil {
					stack := string(debug.Stack())

					var err error
					switch t := errI.(type) {
					case error:
						err = t
					case string:
						err = errors.New(t)
					default:
						err = fmt.Errorf("unknown error: %v", t)
					}
					logrus.WithError(err).Error("Recovered from panic in event handler\n" + stack)
				}
			}()

			err := v.F(data)
			if err != nil {
				logrus.Errorf("An error occurred in a discord event handler: %+v", err)
			}
		}()
	}
}
