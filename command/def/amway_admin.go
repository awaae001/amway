package def

import (
	"github.com/bwmarrin/discordgo"
)

var AmwayAdminCommand = &discordgo.ApplicationCommand{
	Name:        "amway_admin",
	Description: "安利小纸条管理员命令",
	NameLocalizations: &map[discordgo.Locale]string{
		discordgo.ChineseCN: "安利管理",
	},
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "action",
			Description: "执行的操作",
			NameLocalizations: map[discordgo.Locale]string{
				discordgo.ChineseCN: "操作",
			},
			Required: true,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{
					Name:  "打印",
					Value: "print",
				},
				{
					Name:  "删除",
					Value: "delete",
				},
				{
					Name:  "重新发送",
					Value: "resend",
				},
				{
					Name:  "封禁",
					Value: "ban",
				},
				{
					Name:  "解除封禁",
					Value: "lift_ban",
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "input",
			Description: "投稿ID",
			NameLocalizations: map[discordgo.Locale]string{
				discordgo.ChineseCN: "输入",
			},
			Required: false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "user_id",
			Description: "用户ID",
			NameLocalizations: map[discordgo.Locale]string{
				discordgo.ChineseCN: "用户",
			},
			Required:     false,
			Autocomplete: true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "duration",
			Description: "封禁时长 (例如 3d, 72h), 留空则为永久封禁",
			NameLocalizations: map[discordgo.Locale]string{
				discordgo.ChineseCN: "时长",
			},
			Required: false,
		},
	},
}
