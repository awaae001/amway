package def

import (
	"github.com/bwmarrin/discordgo"
)

var CreatePanelCommand = &discordgo.ApplicationCommand{
	Name:        "创建投稿面板",
	Description: "创建一个新的投稿面板",
}
