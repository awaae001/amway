package amway

import (
	"amway/config"
	"amway/db"
	"amway/model"
	"amway/utils"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// stringPtr is a helper function to get a pointer to a string.
func stringPtr(s string) *string {
	return &s
}

// sendPublicationMessage sends the approved submission to the publication channel.
func sendPublicationMessage(s *discordgo.Session, submission *model.Submission) (*discordgo.Message, error) {
	publishChannelID := config.Cfg.AmwayBot.Amway.PublishChannelID
	if publishChannelID == "" {
		return nil, fmt.Errorf("publish channel ID not configured")
	}

	var authorDisplay string
	if submission.IsAnonymous {
		authorDisplay = "一位热心的安利员"
	} else {
		authorDisplay = fmt.Sprintf("<@%s>", submission.UserID)
	}
	plainContent := fmt.Sprintf("-# 来自 %s 的安利\n## %s\n%s",
		authorDisplay,
		submission.RecommendTitle,
		submission.RecommendContent,
	)

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

	publishMsg, err := s.ChannelMessageSendComplex(publishChannelID, &discordgo.MessageSend{
		Content: plainContent,
		Embed:   embed,
	})
	if err != nil {
		return nil, fmt.Errorf("error sending to publish channel: %w", err)
	}

	s.MessageReactionAdd(publishChannelID, publishMsg.ID, "👍")
	s.MessageReactionAdd(publishChannelID, publishMsg.ID, "✅")
	s.MessageReactionAdd(publishChannelID, publishMsg.ID, "❌")

	return publishMsg, nil
}

// sendNotificationToOriginalPost sends a notification to the original post about the submission.
func sendNotificationToOriginalPost(s *discordgo.Session, submission *model.Submission, publishMsg *discordgo.Message) {
	originalChannelID, originalMessageID, err := utils.GetOriginalPostDetails(submission.URL)
	if err != nil {
		log.Printf("Error getting original post details for submission %s: %v", submission.ID, err)
		return
	}

	amwayMessageURL := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", submission.GuildID, publishMsg.ChannelID, publishMsg.ID)

	var authorDisplay string
	if submission.IsAnonymous {
		authorDisplay = "一位热心的安利员"
	} else {
		authorDisplay = fmt.Sprintf("<@%s>", submission.UserID)
	}
	notificationContent := fmt.Sprintf("-# 来自 %s 的推荐，TA 觉得你的帖子很棒！\n## %s\n%s",
		authorDisplay,
		submission.RecommendTitle,
		submission.RecommendContent,
	)

	notificationEmbed := &discordgo.MessageEmbed{
		Title: "安利详情",
		Color: 0x2ea043,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "安利人",
				Value:  authorDisplay,
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

	_, err = s.ChannelMessageSendComplex(originalChannelID, &discordgo.MessageSend{
		Content: notificationContent,
		Embed:   notificationEmbed,
		Reference: &discordgo.MessageReference{
			MessageID: originalMessageID,
			ChannelID: originalChannelID,
			GuildID:   submission.GuildID,
		},
	})
	if err != nil {
		if restErr, ok := err.(*discordgo.RESTError); ok && restErr.Message != nil && restErr.Message.Code == 30033 {
			log.Printf("Skipping notification for submission %s: thread participants limit reached.", submission.ID)
		} else {
			log.Printf("Error sending notification to original post for submission %s: %v", submission.ID, err)
		}
	}
}

// ApproveSubmissionHandler handles approval of submissions
func ApproveSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Immediately acknowledge the interaction to avoid timeout
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		fmt.Printf("Error sending deferred response: %v\n", err)
		return
	}

	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	go func() {
		// Get submission details
		submission, err := db.GetSubmission(submissionID)
		if err != nil {
			fmt.Printf("Error getting submission: %v\n", err)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: stringPtr("无法找到该投稿 "),
			})
			return
		}

		// Update submission status
		err = db.UpdateSubmissionReviewer(submissionID, "approved", i.Member.User.ID)
		if err != nil {
			fmt.Printf("Error updating submission status: %v\n", err)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: stringPtr("更新状态失败 "),
			})
			return
		}

		// Send to publish channel
		publishMsg, err := sendPublicationMessage(s, submission)
		if err != nil {
			fmt.Printf("Error sending publication message for submission %s: %v\n", submissionID, err)
		} else {
			// Update submission with the final message ID
			if err := db.UpdateFinalAmwayMessageID(submissionID, publishMsg.ID); err != nil {
				fmt.Printf("Error updating final amway message ID for submission %s: %v\n", submissionID, err)
			}
			// Send a notification to the original post
			sendNotificationToOriginalPost(s, submission, publishMsg)
		}

		// Update the review message
		originalEmbeds := i.Message.Embeds
		approvedEmbed := &discordgo.MessageEmbed{
			Title:       "审核结果",
			Description: fmt.Sprintf("**审核员:** <@%s>\n**状态:** ✅ 已通过并发布", i.Member.User.ID),
			Color:       0x00FF00,
		}
		updatedEmbeds := append(originalEmbeds, approvedEmbed)

		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds:     &updatedEmbeds,
			Components: &[]discordgo.MessageComponent{}, // Remove buttons
		})
	}()
}

// RejectSubmissionHandler handles rejection of submissions
func RejectSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	err := db.UpdateSubmissionReviewer(submissionID, "rejected", i.Member.User.ID)
	if err != nil {
		fmt.Printf("Error updating submission status: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "更新状态失败 ",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	originalEmbeds := i.Message.Embeds
	rejectedEmbed := &discordgo.MessageEmbed{
		Title:       "审核结果",
		Description: fmt.Sprintf("**审核员:** <@%s>\n**状态:** ❌ 已拒绝", i.Member.User.ID),
		Color:       0xFF0000,
	}
	updatedEmbeds := append(originalEmbeds, rejectedEmbed)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     updatedEmbeds,
			Components: []discordgo.MessageComponent{},
		},
	})
}

// IgnoreSubmissionHandler handles ignoring submissions
func IgnoreSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	err := db.UpdateSubmissionReviewer(submissionID, "ignored", i.Member.User.ID)
	if err != nil {
		fmt.Printf("Error updating submission status: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "更新状态失败 ",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	originalEmbeds := i.Message.Embeds
	ignoredEmbed := &discordgo.MessageEmbed{
		Title:       "审核结果",
		Description: fmt.Sprintf("**审核员:** <@%s>\n**状态:** ⏭️ 已忽略", i.Member.User.ID),
		Color:       0x808080,
	}
	updatedEmbeds := append(originalEmbeds, ignoredEmbed)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     updatedEmbeds,
			Components: []discordgo.MessageComponent{},
		},
	})
}

// BanSubmissionHandler handles banning users and their submissions
func BanSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	// Get submission to get user ID
	submission, err := db.GetSubmission(submissionID)
	if err != nil {
		fmt.Printf("Error getting submission: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "无法找到该投稿 ",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Ban the user
	err = db.BanUser(submission.UserID, "违规投稿")
	if err != nil {
		fmt.Printf("Error banning user: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "封禁用户失败 ",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update submission status
	err = db.UpdateSubmissionReviewer(submissionID, "rejected", i.Member.User.ID)
	if err != nil {
		fmt.Printf("Error updating submission status: %v\n", err)
	}

	originalEmbeds := i.Message.Embeds
	bannedEmbed := &discordgo.MessageEmbed{
		Title:       "审核结果",
		Description: fmt.Sprintf("**审核员:** <@%s>\n**状态:** 🔨 用户已封禁，投稿已拒绝", i.Member.User.ID),
		Color:       0x8B0000,
	}
	updatedEmbeds := append(originalEmbeds, bannedEmbed)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     updatedEmbeds,
			Components: []discordgo.MessageComponent{},
		},
	})
}

// DeleteSubmissionHandler handles deletion of submissions
func DeleteSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	err := db.DeleteSubmission(submissionID)
	if err != nil {
		fmt.Printf("Error deleting submission: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "删除投稿失败 ",
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
