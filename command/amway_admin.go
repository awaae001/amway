package command

import (
	"github.com/bwmarrin/discordgo"
)

var AmwayAdminCommand = &discordgo.ApplicationCommand{
	Name:        "amway_admin",
	Description: "安利小纸条管理员命令",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "action",
			Description: "执行的操作",
			Required:    true,
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
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "input",
			Description: "投稿ID",
			Required:    true,
		},
	},
}