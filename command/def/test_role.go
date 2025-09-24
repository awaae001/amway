package def

import (
	"github.com/bwmarrin/discordgo"
)

var TestAssignRoleCommand = &discordgo.ApplicationCommand{
	Name:        "test_assign_role",
	Description: "测试 gRPC 分配身份组功能",
	NameLocalizations: &map[discordgo.Locale]string{
		discordgo.ChineseCN: "测试分配身份组",
	},
	DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	Options: []*discordgo.ApplicationCommandOption{
		// 服务器是执行命令的服务器，防止滥用
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "config_id",
			Description: "角色配置 ID",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "user",
			Description: "要分配身份组的用户",
			Required:    true,
		},
	},
}
