package events

import "github.com/bwmarrin/discordgo"

// sceduledMessageCreateFunctions serves as a map of all the functions that is run when a user sends a message in a channel the bot is in
var sceduledMessageCreateFunctions []func(g *discordgo.MessageCreate)

// RegisterMessageCreateFunctions adds each message create function to the map of functions ran when a message is created
func RegisterMessageCreateFunctions(funcMap []func(g *discordgo.MessageCreate)) {
	sceduledMessageCreateFunctions = append(sceduledMessageCreateFunctions, funcMap...)
}

// messageCreate is called when any new message is sent in a channel
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	for _, createFunction := range sceduledMessageCreateFunctions {
		go createFunction(m)
	}
}

// sceduledMessageUpdateFunctions serves as a map of all the functions that is run when a user updates a message in a channel the bot is in
var sceduledMessageUpdateFunctions []func(g *discordgo.MessageUpdate)

// RegisterMessageUpdateFunctions adds each message update function to the map of functions ran when a message is updated
func RegisterMessageUpdateFunctions(funcMap []func(g *discordgo.MessageUpdate)) {
	sceduledMessageUpdateFunctions = append(sceduledMessageUpdateFunctions, funcMap...)
}

// messageUpdate is sent when a message is updated
func messageUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) {
	for _, updateFunction := range sceduledMessageUpdateFunctions {
		go updateFunction(m)
	}
}

// sceduledMessageDeleteFunctions serves as a map of all the functions that is run when a user deletes a message in a channel the bot is in
var sceduledMessageDeleteFunctions []func(g *discordgo.MessageDelete)

// RegisterMessageDeleteFunctions adds each message delete function to the map of functions ran when a message is deleted
func RegisterMessageDeleteFunctions(funcMap []func(g *discordgo.MessageDelete)) {
	sceduledMessageDeleteFunctions = append(sceduledMessageDeleteFunctions, funcMap...)
}

// messageDelete is sent when a message is deleted
func messageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	for _, deleteFunction := range sceduledMessageDeleteFunctions {
		go deleteFunction(m)
	}
}
