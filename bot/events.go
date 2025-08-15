package bot

import (
	"amway/handler"
	"amway/handler/amway"

	"github.com/bwmarrin/discordgo"
)

func registerEventHandlers(s *discordgo.Session) {
	s.AddHandler(handler.OnInteractionCreate)
	s.AddHandler(amway.MessageReactionAdd)
	s.AddHandler(amway.MessageReactionRemove)
	s.AddHandler(amway.MessageCreate)

	// 设置必要的intents
	s.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds | discordgo.IntentsGuildMessageReactions
}
