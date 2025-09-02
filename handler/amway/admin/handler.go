package amway_admin

import (
	"amway/utils"
	"log"

	"github.com/bwmarrin/discordgo"
)

// AmwayAdminCommandHandler handles the /amway_admin command
func AmwayAdminCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {

	// 立即响应交互
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral, // 仅管理员可见
		},
	})
	if err != nil {
		log.Printf("Error sending deferred response: %v", err)
		return
	}

	// 在 goroutine 中处理后续逻辑
	go func() {
		// 权限检查：只有管理员才能使用此命令
		if !utils.CheckAuth(i.Member.User.ID, i.Member.Roles) {
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr("❌ 您没有权限执行此操作 "),
			})
			return
		}

		// 获取命令参数
		options := i.ApplicationCommandData().Options
		var action, input, userID, duration string

		for _, option := range options {
			switch option.Name {
			case "action":
				action = option.StringValue()
			case "input":
				input = option.StringValue()
			case "user_id":
				userID = option.StringValue()
			case "duration":
				duration = option.StringValue()
			}
		}

		// 根据action执行相应操作
		switch action {
		case "print":
			handlePrintSubmission(s, i, input)
		case "delete":
			handleDeleteSubmission(s, i, input)
		case "resend":
			handleResendSubmission(s, i, input)
		case "ban":
			handleBanUser(s, i, userID, duration)
		case "lift_ban":
			handleLiftBan(s, i, userID)
		default:
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr("❌ 未知的操作类型 "),
			})
		}
	}()
}
