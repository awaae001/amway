package amway

import (
	"amway/config"
	"amway/utils"
	"context"
	"fmt"
	"log"
	"time"

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

	go func() {
		// 设置超时上下文，防止 goroutine 长时间运行
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in panel creation goroutine: %v", r)
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: utils.StringPtr("创建面板时发生内部错误。"),
				})
			}
		}()

		// 检查超时
		select {
		case <-ctx.Done():
			log.Printf("Panel creation timed out")
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr("创建面板超时，请稍后重试。"),
			})
			return
		default:
		}
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

		// 发送到目标频道
		message, err := s.ChannelMessageSendComplex(channelID, CreatePanelMessage())

		// 3. 根据结果编辑原始的延迟响应。
		if err != nil {
			log.Printf("Error sending panel message: %v", err)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr(fmt.Sprintf("创建面板时出错：%v", err)),
			})
			return
		}

		// 保存面板状态到JSON文件
		if err := utils.SavePanelState("data/panel_state.json", channelID, message.ID); err != nil {
			log.Printf("Error saving panel state: %v", err)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr(fmt.Sprintf("创建面板成功，但保存状态失败：%v", err)),
			})
			return
		}

		// 成功
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr("✅ 投稿面板已成功创建！"),
		})
	}()
}

// MessageCreate 监听新消息并更新面板
func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// 加载面板状态
	panelState, err := utils.LoadPanelState("data/panel_state.json")
	if err != nil {
		log.Printf("Error loading panel state: %v", err)
		return
	}

	// 如果没有面板状态，不做任何处理
	if panelState == nil {
		return
	}

	// 检查消息是否来自面板所在的频道
	if m.ChannelID != panelState.ChannelID {
		return
	}

	// 检查消息是否为机器人自己发送的面板消息，以防止递归
	// 通过检查 Embed 的标题来精确识别面板消息
	if m.Author.ID == s.State.User.ID {
		if len(m.Embeds) > 0 && m.Embeds[0].Title == "鉴赏家投稿面板" {
			log.Printf("Ignoring bot's own panel message %s to prevent recursion.", m.ID)
			return
		}
	}
	// 删除旧的面板消息
	if err := s.ChannelMessageDelete(panelState.ChannelID, panelState.MessageID); err != nil {
		log.Printf("Error deleting old panel message: %v", err)
	}

	// 发送新的面板消息
	newMessage, err := s.ChannelMessageSendComplex(panelState.ChannelID, CreatePanelMessage())

	if err != nil {
		log.Printf("Error sending new panel message: %v", err)
		return
	}

	// 更新面板状态
	if err := utils.SavePanelState("data/panel_state.json", panelState.ChannelID, newMessage.ID); err != nil {
		log.Printf("Error saving new panel state: %v", err)
	}

	log.Printf("Panel updated due to new message in channel %s", m.ChannelID)
}
