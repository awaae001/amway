package def

import "github.com/bwmarrin/discordgo"

var LookupCommand = &discordgo.ApplicationCommand{
	Name:        "lookup",
	Description: "查询用户的投稿历史",
	NameLocalizations: &map[discordgo.Locale]string{
		discordgo.ChineseCN: "查询投稿",
	},
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "user",
			Description: "要查询的用户 (默认为自己)",
			NameLocalizations: map[discordgo.Locale]string{
				discordgo.ChineseCN: "用户",
			},
			Required: false,
		},
	},
}
