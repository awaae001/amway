package command

import (
	"amway/command/def"

	"github.com/bwmarrin/discordgo"
)

// AllCommands 包含所有命令
var AllCommands = []*discordgo.ApplicationCommand{
	def.AmwayAdminCommand,
	def.CreatePanelCommand,
	def.LookupCommand,
	def.RebuildCommand,
	def.TestAssignRoleCommand,
}
