package amway

import (
	"amway/config"
	"amway/db"
	"amway/handler/tools"
	"amway/model"
	"amway/utils"
	"amway/vote"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

// BuildVoteStatusEmbed builds the embed for the current voting status.
func BuildVoteStatusEmbed(session *vote.Session) *discordgo.MessageEmbed {
	var voteSummary string
	for _, v := range session.Votes {
		if (v.Type == vote.Reject || v.Type == vote.Ban) && v.Reason != "" {
			voteSummary += fmt.Sprintf("<@%s>投了 `%s`\n> 理由: %s\n", v.VoterID, v.Type, v.Reason)
		} else {
			voteSummary += fmt.Sprintf("<@%s>投了 `%s`\n", v.VoterID, v.Type)
		}
	}

	voteEmbed := &discordgo.MessageEmbed{
		Title:       "当前投票状态",
		Description: voteSummary,
		Color:       0x00BFFF, // Deep sky blue
	}

	if len(session.Votes) == 2 {
		voteCounts := make(map[vote.VoteType]int)
		for _, v := range session.Votes {
			voteCounts[v.Type]++
			if v.Type == vote.Feature {
				voteCounts[vote.Pass]++
			}
		}
		if !tools.HasConsensus(voteCounts) {
			voteEmbed.Fields = []*discordgo.MessageEmbedField{
				{
					Name:  "注意",
					Value: "前两票出现差异，等待第三票决定最终结果",
				},
			}
		}
	}
	return voteEmbed
}

// BuildFinalVoteEmbed builds the embed for the completed vote.
func BuildFinalVoteEmbed(submissionID, finalStatus string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "✅ 投票结束",
		Description: fmt.Sprintf("对投稿 `%s` 的投票已完成\n\n**最终结果:** `%s`", submissionID, finalStatus),
		Color:       0x5865F2, // Discord Blurple
	}
}

// BuildRejectionComponents builds the buttons for sending rejection reasons.
func BuildRejectionComponents(cacheID string, reasons []string) []discordgo.MessageComponent {
	var components []discordgo.MessageComponent
	if len(reasons) > 0 {
		reasonButtons := []discordgo.MessageComponent{}
		for idx := range reasons {
			reasonButtons = append(reasonButtons, discordgo.Button{
				Label:    fmt.Sprintf("理由%d", idx+1),
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("select_reason:%s:%d", cacheID, idx),
			})
		}

		const maxButtonsPerRow = 5
		for i := 0; i < len(reasonButtons); i += maxButtonsPerRow {
			end := i + maxButtonsPerRow
			if end > len(reasonButtons) {
				end = len(reasonButtons)
			}
			components = append(components, discordgo.ActionsRow{
				Components: reasonButtons[i:end],
			})
		}

		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "发送私信通知",
					Style:    discordgo.PrimaryButton,
					CustomID: "send_rejection_dm:" + cacheID,
				},
			},
		})
	}
	return components
}

// BuildBanComponents builds the buttons for sending ban reasons.
func BuildBanComponents(cacheID string, reasons []string) []discordgo.MessageComponent {
	var components []discordgo.MessageComponent
	if len(reasons) > 0 {
		var reasonButtons []discordgo.MessageComponent
		for idx, reason := range reasons {
			// Truncate reason for button label if it's too long
			label := reason
			if len(label) > 20 {
				label = label[:17] + "..."
			}
			reasonButtons = append(reasonButtons, discordgo.Button{
				Label:    label,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("select_ban_reason:%s:%d", cacheID, idx),
			})
		}

		const maxButtonsPerRow = 5
		for i := 0; i < len(reasonButtons); i += maxButtonsPerRow {
			end := i + maxButtonsPerRow
			if end > len(reasonButtons) {
				end = len(reasonButtons)
			}
			components = append(components, discordgo.ActionsRow{
				Components: reasonButtons[i:end],
			})
		}

		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "发送封禁通知",
					Style:    discordgo.DangerButton,
					CustomID: "send_ban_dm:" + cacheID,
				},
			},
		})
	}
	return components
}

// getAuthorDisplay returns the appropriate author display string based on anonymity status.
func getAuthorDisplay(submission *model.Submission) string {
	if submission.IsAnonymous {
		return "一位热心的安利员"
	}
	return fmt.Sprintf("<@%s>", submission.UserID)
}

// formatRecommendationContent formats the recommendation content with author and title.
func formatRecommendationContent(submission *model.Submission, prefix string) string {
	authorDisplay := getAuthorDisplay(submission)
	return fmt.Sprintf("-# %s %s\n## %s\n%s",
		prefix,
		authorDisplay,
		submission.RecommendTitle,
		submission.RecommendContent,
	)
}

// BuildPublicationMessage constructs the message for the publication channel.
func BuildPublicationMessage(submission *model.Submission) (*discordgo.MessageSend, error) {
	if config.Cfg.AmwayBot.Amway.PublishChannelID == "" {
		return nil, fmt.Errorf("publish channel ID not configured")
	}

	plainContent := formatRecommendationContent(submission, "来自")

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

	return &discordgo.MessageSend{
		Content: plainContent,
		Embed:   embed,
	}, nil
}

// BuildNotificationMessage constructs the notification message to be sent to the original post.
func BuildNotificationMessage(submission *model.Submission, publishMsg *discordgo.Message) (string, *discordgo.MessageSend, error) {
	originalChannelID, originalMessageID, err := utils.GetOriginalPostDetails(submission.URL)
	if err != nil {
		return "", nil, fmt.Errorf("error getting original post details for submission %s: %w", submission.ID, err)
	}

	amwayMessageURL := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", submission.GuildID, publishMsg.ChannelID, publishMsg.ID)

	authorDisplay := getAuthorDisplay(submission)
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
				Value:  getAuthorDisplay(submission),
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

	messageSend := &discordgo.MessageSend{
		Content: notificationContent,
		Embed:   notificationEmbed,
		Reference: &discordgo.MessageReference{
			MessageID: originalMessageID,
			ChannelID: originalChannelID,
			GuildID:   submission.GuildID,
		},
	}

	return originalChannelID, messageSend, nil
}

// PublishSubmission handles the entire process of publishing an approved or featured submission.
func PublishSubmission(s *discordgo.Session, submission *model.Submission, replyToOriginal bool) {
	publicationMessage, err := BuildPublicationMessage(submission)
	if err != nil {
		log.Printf("Error building publication message for submission %s: %v", submission.ID, err)
		return
	}

	publishMsg, err := s.ChannelMessageSendComplex(config.Cfg.AmwayBot.Amway.PublishChannelID, publicationMessage)
	if err != nil {
		log.Printf("Error sending publication message for submission %s: %v", submission.ID, err)
		return
	}

	// Add standard reactions to the published message
	s.MessageReactionAdd(publishMsg.ChannelID, publishMsg.ID, "👍")
	s.MessageReactionAdd(publishMsg.ChannelID, publishMsg.ID, "🤔")
	s.MessageReactionAdd(publishMsg.ChannelID, publishMsg.ID, "🚫")

	if err := db.UpdateFinalAmwayMessageID(submission.ID, publishMsg.ID); err != nil {
		log.Printf("Error updating final amway message ID for submission %s: %v", submission.ID, err)
	}

	if replyToOriginal {
		sendNotificationToOriginalPost(s, submission, publishMsg)
	}
}

// sendNotificationToOriginalPost sends a notification to the original post about the submission.
func sendNotificationToOriginalPost(s *discordgo.Session, submission *model.Submission, publishMsg *discordgo.Message) {
	originalChannelID, notification, err := BuildNotificationMessage(submission, publishMsg)
	if err != nil {
		log.Printf("Error building notification message for submission %s: %v", submission.ID, err)
		return
	}

	msg, err := s.ChannelMessageSendComplex(originalChannelID, notification)
	if err != nil {
		if restErr, ok := err.(*discordgo.RESTError); ok && restErr.Message != nil && restErr.Message.Code == 30033 {
			log.Printf("Skipping notification for submission %s: thread participants limit reached.", submission.ID)
		} else {
			log.Printf("Error sending notification to original post for submission %s: %v", submission.ID, err)
		}
		return
	}

	if err := db.UpdateThreadMessageID(submission.ID, msg.ID); err != nil {
		log.Printf("Error updating thread message ID for submission %s: %v", submission.ID, err)
	}
}

// UpdateNotificationInOriginalPost updates an existing notification message in the original post.
// This is used when the submission's anonymity status changes or when the submission is deleted.
func UpdateNotificationInOriginalPost(s *discordgo.Session, submission *model.Submission, publishMsg *discordgo.Message, isDeleted bool) error {
	originalChannelID, _, err := utils.GetOriginalPostDetails(submission.URL)
	if err != nil {
		return fmt.Errorf("error getting original post details for submission %s: %w", submission.ID, err)
	}

	// Check if thread message exists
	if submission.ThreadMessageID == "" || submission.ThreadMessageID == "0" {
		return nil // No message to update
	}

	var notificationContent string
	var notificationEmbed *discordgo.MessageEmbed

	if isDeleted {
		// Build deleted message
		notificationContent = "-# 该安利已被作者删除"
		notificationEmbed = &discordgo.MessageEmbed{
			Title: "安利已删除",
			Color: 0xFF0000, // Red color for deletion
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "状态",
					Value: "该推荐已被作者删除",
				},
				{
					Name:   "时间",
					Value:  time.Unix(submission.Timestamp, 0).Format("2006-01-02 15:04:05"),
					Inline: true,
				},
			},
		}
	} else {
		// Build updated message with current anonymity status
		authorDisplay := getAuthorDisplay(submission)
		notificationContent = fmt.Sprintf("-# 来自 %s 的推荐，TA 觉得你的帖子很棒！\n## %s\n%s",
			authorDisplay,
			submission.RecommendTitle,
			submission.RecommendContent,
		)

		embedFields := []*discordgo.MessageEmbedField{
			{
				Name:   "安利人",
				Value:  getAuthorDisplay(submission),
				Inline: true,
			},
			{
				Name:   "时间",
				Value:  time.Unix(submission.Timestamp, 0).Format("2006-01-02 15:04:05"),
				Inline: true,
			},
		}

		// Add amway message link if available
		if publishMsg != nil {
			amwayMessageURL := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", submission.GuildID, publishMsg.ChannelID, publishMsg.ID)
			embedFields = append(embedFields, &discordgo.MessageEmbedField{
				Name:  "安利消息链接",
				Value: fmt.Sprintf("[点击查看](%s)", amwayMessageURL),
			})
		}

		notificationEmbed = &discordgo.MessageEmbed{
			Title:  "安利详情",
			Color:  0x2ea043,
			Fields: embedFields,
		}
	}

	// Update the message
	_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: originalChannelID,
		ID:      submission.ThreadMessageID,
		Content: &notificationContent,
		Embeds:  &[]*discordgo.MessageEmbed{notificationEmbed},
	})

	if err != nil {
		return fmt.Errorf("error updating notification message for submission %s: %w", submission.ID, err)
	}

	return nil
}
