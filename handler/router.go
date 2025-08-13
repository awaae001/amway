package handler

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	commandHandlers   = make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))
	componentHandlers = make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))
	modalHandlers     = make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))
)

// AddCommandHandler registers a handler for a slash command.
func AddCommandHandler(name string, handler func(s *discordgo.Session, i *discordgo.InteractionCreate)) {
	commandHandlers[name] = handler
}

// AddComponentHandler registers a handler for a message component.
func AddComponentHandler(customID string, handler func(s *discordgo.Session, i *discordgo.InteractionCreate)) {
	componentHandlers[customID] = handler
}

// AddModalHandler registers a handler for a modal submission.
func AddModalHandler(customID string, handler func(s *discordgo.Session, i *discordgo.InteractionCreate)) {
	modalHandlers[customID] = handler
}

// OnInteractionCreate is the main interaction router.
// It should be registered as the primary interaction handler in main.go.
func OnInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if handler, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			handler(s, i)
		}
	case discordgo.InteractionMessageComponent:
		customID := i.MessageComponentData().CustomID
		parts := strings.SplitN(customID, ":", 2)
		handlerKey := parts[0]

		if handler, ok := componentHandlers[handlerKey]; ok {
			handler(s, i)
		}
	case discordgo.InteractionModalSubmit:
		if handler, ok := modalHandlers[i.ModalSubmitData().CustomID]; ok {
			handler(s, i)
		}
	}
}
