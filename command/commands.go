package command

import (
	"amway/command/def"

	"github.com/bwmarrin/discordgo"
)

// AllCommands contains all of the commands
var AllCommands = []*discordgo.ApplicationCommand{
	def.AmwayAdminCommand,
	def.CreatePanelCommand,
	def.LookupCommand,
}
