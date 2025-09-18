package def

import (
	"github.com/bwmarrin/discordgo"
)

var RebuildCommand = &discordgo.ApplicationCommand{
	Name:        "rebuild",
	Description: "重建丢失缓存的pending安利（48小时内）",
	NameLocalizations: &map[discordgo.Locale]string{
		discordgo.ChineseCN: "重建",
	},
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionBoolean,
			Name:        "dry_run",
			Description: "仅显示会被重建的安利数量，不实际执行",
			NameLocalizations: map[discordgo.Locale]string{
				discordgo.ChineseCN: "预览模式",
			},
			Required: false,
		},
	},
}