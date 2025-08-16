package amway

import (
	"amway/config"
	"amway/model"
	"amway/utils"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// CreatePanelMessage 创建标准的投稿面板消息
func CreatePanelMessage() *discordgo.MessageSend {
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

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{button}},
		},
	}
}

// approveSubmissionHandler handles approval of submissions
func ApproveSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	// Get submission details
	submission, err := utils.GetSubmission(submissionID)
	if err != nil {
		fmt.Printf("Error getting submission: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "无法找到该投稿。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update submission status
	err = utils.UpdateSubmissionStatus(submissionID, "approved")
	if err != nil {
		fmt.Printf("Error updating submission status: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "更新状态失败。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Send to publish channel with new format
	publishChannelID := config.Cfg.AmwayBot.Amway.PublishChannelID
	if publishChannelID != "" {
		// 构建新的发布格式
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
			Color:  0x2ea043, // 与Discord深色主题背景色一致
			Fields: embedFields,
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("安利提交于: %s • ID: %s", time.Unix(submission.Timestamp, 0).Format("2006-01-02 15:04:05"), submission.ID),
			},
		}

		publishMsg, err := s.ChannelMessageSendComplex(publishChannelID, &discordgo.MessageSend{
			Content: plainContent,
			Embed:   embed,
		})
		if err != nil {
			fmt.Printf("Error sending to publish channel: %v\n", err)
		} else {
			// Add reactions
			s.MessageReactionAdd(publishChannelID, publishMsg.ID, "👍")
			s.MessageReactionAdd(publishChannelID, publishMsg.ID, "❓")
			s.MessageReactionAdd(publishChannelID, publishMsg.ID, "👎")

			// Update submission with the final message ID
			err := utils.UpdateFinalAmwayMessageID(submissionID, publishMsg.ID)
			if err != nil {
				fmt.Printf("Error updating final amway message ID: %v\n", err)
			}

			// Send a notification to the original post
			originalChannelID, originalMessageID, err := utils.GetOriginalPostDetails(submission.URL)
			if err != nil {
				fmt.Printf("Error getting original post details: %v\n", err)
			} else {
				amwayMessageURL := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", submission.GuildID, publishChannelID, publishMsg.ID)

				notificationContent := fmt.Sprintf("-# 来自 <@%s> 的推荐，TA 觉得你的帖子很棒！\n## %s\n%s",
					submission.UserID,
					submission.RecommendTitle,
					submission.RecommendContent,
				)

				notificationEmbed := &discordgo.MessageEmbed{
					Title: "安利详情",
					Color: 0x2ea043,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "安利人",
							Value:  fmt.Sprintf("<@%s>", submission.UserID),
							Inline: true,
						},
						{
							Name:   "时间",
							Value:  time.Unix(submission.Timestamp, 0).Format("2006-01-02 15:04:05"),
							Inline: true,
						},
						{
							Name:  "安利消息链接",
							Value: fmt.Sprintf("[点击查看](%s)", amwayMessageURL),
						},
					},
				}

				_, err := s.ChannelMessageSendComplex(originalChannelID, &discordgo.MessageSend{
					Content: notificationContent,
					Embed:   notificationEmbed,
					Reference: &discordgo.MessageReference{
						MessageID: originalMessageID,
						ChannelID: originalChannelID,
						GuildID:   submission.GuildID,
					},
				})
				if err != nil {
					fmt.Printf("Error sending notification to original post: %v\n", err)
				}
			}
		}
	}

	// Update the review message
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "投稿已通过",
					Description: fmt.Sprintf("**投稿ID:** %s\n**审核员:** <@%s>\n**状态:** ✅ 已通过并发布", submissionID, i.Member.User.ID),
					Color:       0x00FF00,
				},
			},
			Components: []discordgo.MessageComponent{}, // Remove buttons
		},
	})
}

// rejectSubmissionHandler handles rejection of submissions
func RejectSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	err := utils.UpdateSubmissionStatus(submissionID, "rejected")
	if err != nil {
		fmt.Printf("Error updating submission status: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "更新状态失败。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "投稿已拒绝",
					Description: fmt.Sprintf("**投稿ID:** %s\n**审核员:** <@%s>\n**状态:** ❌ 已拒绝", submissionID, i.Member.User.ID),
					Color:       0xFF0000,
				},
			},
			Components: []discordgo.MessageComponent{},
		},
	})
}

// ignoreSubmissionHandler handles ignoring submissions
func IgnoreSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	err := utils.UpdateSubmissionStatus(submissionID, "ignored")
	if err != nil {
		fmt.Printf("Error updating submission status: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "更新状态失败。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "投稿已忽略",
					Description: fmt.Sprintf("**投稿ID:** %s\n**审核员:** <@%s>\n**状态:** ⏭️ 已忽略", submissionID, i.Member.User.ID),
					Color:       0x808080,
				},
			},
			Components: []discordgo.MessageComponent{},
		},
	})
}

// banSubmissionHandler handles banning users and their submissions
func BanSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	// Get submission to get user ID
	submission, err := utils.GetSubmission(submissionID)
	if err != nil {
		fmt.Printf("Error getting submission: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "无法找到该投稿。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Ban the user
	err = utils.BanUser(submission.UserID, "违规投稿")
	if err != nil {
		fmt.Printf("Error banning user: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "封禁用户失败。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update submission status
	err = utils.UpdateSubmissionStatus(submissionID, "rejected")
	if err != nil {
		fmt.Printf("Error updating submission status: %v\n", err)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "用户已封禁",
					Description: fmt.Sprintf("**投稿ID:** %s\n**审核员:** <@%s>\n**状态:** 🔨 用户已封禁，投稿已拒绝", submissionID, i.Member.User.ID),
					Color:       0x8B0000,
				},
			},
			Components: []discordgo.MessageComponent{},
		},
	})
}

// deleteSubmissionHandler handles deletion of submissions
func DeleteSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	err := utils.DeleteSubmission(submissionID)
	if err != nil {
		fmt.Printf("Error deleting submission: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "删除投稿失败。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "投稿已删除",
					Description: fmt.Sprintf("**投稿ID:** %s\n**审核员:** <@%s>\n**状态:** 🗑️ 已删除", submissionID, i.Member.User.ID),
					Color:       0x000000,
				},
			},
			Components: []discordgo.MessageComponent{},
		},
	})
}

// SendSubmissionToReviewChannel sends a submission to the review channel with appropriate formatting.
func SendSubmissionToReviewChannel(s *discordgo.Session, submission *model.Submission) {
	reviewChannelID := config.Cfg.AmwayBot.Amway.ReviewChannelID
	if reviewChannelID == "" {
		fmt.Println("Review channel ID not configured")
		return
	}

	var embed *discordgo.MessageEmbed
	// Differentiate between legacy and new submissions based on RecommendTitle
	if submission.RecommendTitle == "" {
		// Legacy submission format
		embed = &discordgo.MessageEmbed{
			Title:       "新的投稿待审核",
			Description: fmt.Sprintf("**投稿ID:** %s\n**作者:** <@%s>\n**标题:** %s\n**URL:** %s\n**内容:**\n%s", submission.ID, submission.UserID, submission.OriginalTitle, submission.URL, submission.RecommendContent),
			Color:       0xFFFF00, // Yellow for pending
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("提交时间 • ID: %s", submission.ID),
			},
		}
	} else {
		// New (V2) submission format
		embed = &discordgo.MessageEmbed{
			Title:       "新的安利投稿待审核",
			Description: fmt.Sprintf("**投稿ID:** %s\n**投稿人:** <@%s>\n**安利标题:** %s\n**原帖作者:** <@%s>\n**原帖链接:** %s\n**安利内容:**\n%s", submission.ID, submission.UserID, submission.RecommendTitle, submission.OriginalAuthor, submission.URL, submission.RecommendContent),
			Color:       0xFFFF00, // Yellow for pending
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("提交时间 • ID: %s", submission.ID),
			},
		}
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "通过",
					Style:    discordgo.SuccessButton,
					CustomID: "approve_submission:" + submission.ID,
					Emoji:    &discordgo.ComponentEmoji{Name: "✅"},
				},
				discordgo.Button{
					Label:    "拒绝",
					Style:    discordgo.DangerButton,
					CustomID: "reject_submission:" + submission.ID,
					Emoji:    &discordgo.ComponentEmoji{Name: "❌"},
				},
				discordgo.Button{
					Label:    "忽略",
					Style:    discordgo.SecondaryButton,
					CustomID: "ignore_submission:" + submission.ID,
					Emoji:    &discordgo.ComponentEmoji{Name: "⏭️"},
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "封禁",
					Style:    discordgo.DangerButton,
					CustomID: "ban_submission:" + submission.ID,
					Emoji:    &discordgo.ComponentEmoji{Name: "🔨"},
				},
				discordgo.Button{
					Label:    "删除",
					Style:    discordgo.DangerButton,
					CustomID: "delete_submission:" + submission.ID,
					Emoji:    &discordgo.ComponentEmoji{Name: "🗑️"},
				},
			},
		},
	}

	_, err := s.ChannelMessageSendComplex(reviewChannelID, &discordgo.MessageSend{
		Embed:      embed,
		Components: components,
	})

	if err != nil {
		fmt.Printf("Error sending review message: %v\n", err)
	}
}
