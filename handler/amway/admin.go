package amway

import (
	"amway/config"
	"amway/utils"
	"fmt"
	"log"
	"time"

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
				Content: utils.StringPtr("❌ 您没有权限执行此操作。"),
			})
			return
		}

		// 获取命令参数
		options := i.ApplicationCommandData().Options
		var action, input string

		for _, option := range options {
			switch option.Name {
			case "action":
				action = option.StringValue()
			case "input":
				input = option.StringValue()
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
		default:
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr("❌ 未知的操作类型。"),
			})
		}
	}()
}

// handlePrintSubmission 打印投稿元数据
func handlePrintSubmission(s *discordgo.Session, i *discordgo.InteractionCreate, submissionID string) {
	submission, err := utils.GetSubmissionWithDeleted(submissionID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(fmt.Sprintf("❌ 获取投稿信息失败：%v", err)),
		})
		return
	}

	if submission == nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(fmt.Sprintf("❌ 未找到ID为 %s 的投稿。", submissionID)),
		})
		return
	}

	// 检查是否已删除
	isDeleted, _ := utils.IsSubmissionDeleted(submissionID)
	deletedStatus := ""
	if isDeleted {
		deletedStatus = " **[已删除]**"
	}

	// 格式化时间
	timestamp := time.Unix(submission.Timestamp, 0)
	timeStr := timestamp.Format("2006-01-02 15:04:05")

	// 构建详细信息
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📋 投稿元数据 - ID: %s%s", submissionID, deletedStatus),
		Color: 0x3498db,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🆔 投稿ID",
				Value:  submission.ID,
				Inline: true,
			},
			{
				Name:   "👤 作者ID",
				Value:  submission.UserID,
				Inline: true,
			},
			{
				Name:   "📝 作者昵称",
				Value:  submission.AuthorNickname,
				Inline: true,
			},
			{
				Name:   "🕒 创建时间",
				Value:  timeStr,
				Inline: true,
			},
			{
				Name:   "🏠 服务器ID",
				Value:  submission.GuildID,
				Inline: true,
			},
			{
				Name:   "🔗 原帖URL",
				Value:  submission.URL,
				Inline: false,
			},
		},
	}

	// 如果有原始帖子信息，添加相关字段
	if submission.OriginalTitle != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "📰 原帖标题",
			Value:  submission.OriginalTitle,
			Inline: false,
		})
	}

	if submission.OriginalAuthor != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "✍️ 原帖作者",
			Value:  submission.OriginalAuthor,
			Inline: true,
		})
	}

	if submission.OriginalPostTimestamp != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "📅 原帖时间",
			Value:  submission.OriginalPostTimestamp,
			Inline: true,
		})
	}

	if submission.RecommendTitle != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "💡 推荐标题",
			Value:  submission.RecommendTitle,
			Inline: false,
		})
	}

	// 反应统计
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "📊 反应统计",
		Value:  fmt.Sprintf("👍 %d | ❓ %d | 👎 %d", submission.Upvotes, submission.Questions, submission.Downvotes),
		Inline: false,
	})

	if submission.FinalAmwayMessageID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🔗 发布消息ID",
			Value:  submission.FinalAmwayMessageID,
			Inline: true,
		})
	}

	// 推荐内容（截断显示）
	if submission.RecommendContent != "" {
		content := submission.RecommendContent
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "💭 推荐内容",
			Value:  content,
			Inline: false,
		})
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

// handleDeleteSubmission 删除（标记）投稿
func handleDeleteSubmission(s *discordgo.Session, i *discordgo.InteractionCreate, submissionID string) {
	// 首先检查投稿是否存在
	submission, err := utils.GetSubmissionWithDeleted(submissionID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(fmt.Sprintf("❌ 获取投稿信息失败：%v", err)),
		})
		return
	}

	if submission == nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(fmt.Sprintf("❌ 未找到ID为 %s 的投稿。", submissionID)),
		})
		return
	}

	// 检查是否已经删除
	isDeleted, err := utils.IsSubmissionDeleted(submissionID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(fmt.Sprintf("❌ 检查删除状态失败：%v", err)),
		})
		return
	}

	if isDeleted {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(fmt.Sprintf("ℹ️ 投稿 %s 已经被标记为删除。", submissionID)),
		})
		return
	}

	// 标记为删除
	err = utils.MarkSubmissionDeleted(submissionID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(fmt.Sprintf("❌ 删除投稿失败：%v", err)),
		})
		return
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: utils.StringPtr(fmt.Sprintf("✅ 投稿 %s 已成功标记为删除。", submissionID)),
	})
}

// handleResendSubmission 重新发送投稿
func handleResendSubmission(s *discordgo.Session, i *discordgo.InteractionCreate, submissionID string) {
	// 获取投稿信息（包括已删除的）
	submission, err := utils.GetSubmissionWithDeleted(submissionID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(fmt.Sprintf("❌ 获取投稿信息失败：%v", err)),
		})
		return
	}

	if submission == nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(fmt.Sprintf("❌ 未找到ID为 %s 的投稿。", submissionID)),
		})
		return
	}

	// 获取发布频道配置
	publishChannelID := config.Cfg.AmwayBot.Amway.PublishChannelID
	if publishChannelID == "" {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr("❌ 配置错误：未设置发布频道 ID。"),
		})
		return
	}

	// 构建发布消息（与原始发布逻辑保持一致）
	// 上半部分：纯文本内容
	plainContent := fmt.Sprintf("-# 来自 <@%s> 的安利\n## %s\n%s",
		submission.UserID,
		submission.RecommendTitle,
		submission.RecommendContent,
	)

	// 下半部分：嵌入式卡片
	embedFields := []*discordgo.MessageEmbedField{
		{
			Name:   "作者",
			Value:  fmt.Sprintf("<@%s>", submission.OriginalAuthor),
			Inline: true,
		},
		{
			Name:   "帖子链接",
			Value:  fmt.Sprintf("[%s](%s)", submission.OriginalTitle, submission.URL),
			Inline: true,
		},
	}

	if submission.OriginalPostTimestamp != "" {
		embedFields = append(embedFields, &discordgo.MessageEmbedField{
			Name:   "发帖日期",
			Value:  submission.OriginalPostTimestamp,
			Inline: true,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:  "详情信息",
		Color:  0x2ea043,
		Fields: embedFields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("安利提交于: %s • ID: %s", time.Unix(submission.Timestamp, 0).Format("2006-01-02 15:04:05"), submission.ID),
		},
	}

	// 发送消息
	message, err := s.ChannelMessageSendComplex(publishChannelID, &discordgo.MessageSend{
		Content: plainContent,
		Embed:   embed,
	})

	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(fmt.Sprintf("❌ 重新发送失败：%v", err)),
		})
		return
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: utils.StringPtr(fmt.Sprintf("✅ 投稿 %s 已成功重新发送到 <#%s>。\n消息链接：https://discord.com/channels/%s/%s/%s",
			submissionID, publishChannelID, submission.GuildID, publishChannelID, message.ID)),
	})
}
