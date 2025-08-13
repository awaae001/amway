package amway

import (
	"amway/config"
	"amway/utils"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func createPanelCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 1. 立即响应交互，告诉 Discord 我们收到了请求。
	// 这必须在 3 秒内完成。我们使用延迟响应。
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral, // 仅发送者可见
		},
	})
	if err != nil {
		log.Printf("Error sending deferred response: %v", err)
		return
	}

	// 2. 将所有后续处理移入一个新的 goroutine 中。
	// 这可以防止任何阻塞操作（如权限检查、数据库、API 调用）影响机器人网关的响应。
	go func() {
		// 权限检查
		if !utils.CheckAuth(i.Member.User.ID, i.Member.Roles) {
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr("您没有权限执行此操作。"),
			})
			return
		}

		// 获取配置
		channelID := config.Cfg.AmwayBot.Amway.PublishChannelID
		if channelID == "" {
			log.Println("Error: PublishChannelID is not configured")
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr("配置错误：未设置发布频道 ID。"),
			})
			return
		}

		// 创建面板消息
		embed := &discordgo.MessageEmbed{
			Title:       "鉴赏家投稿面板",
			Description: "点击下方按钮开始投稿您的简评",
			Color:       0x5865F2, // Discord Blurple
		}
		button := discordgo.Button{
			Label:    "点击投稿",
			Style:    discordgo.PrimaryButton,
			CustomID: "create_submission_button",
			Emoji:    &discordgo.ComponentEmoji{Name: "📝"},
		}

		// 发送到目标频道
		_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{button}},
			},
		})

		// 3. 根据结果编辑原始的延迟响应。
		if err != nil {
			log.Printf("Error sending panel message: %v", err)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr(fmt.Sprintf("创建面板时出错：%v", err)),
			})
			return
		}

		// 成功
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr("✅ 投稿面板已成功创建！"),
		})
	}()
}
