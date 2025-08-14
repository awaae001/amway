package amway

import (
	"amway/config"
	"amway/utils"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

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
				Value:  fmt.Sprintf("[点击跳转](%s)", submission.URL),
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
