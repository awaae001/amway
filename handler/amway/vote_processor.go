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

// processVote is the core logic for handling a vote submission.
func processVote(s *discordgo.Session, i *discordgo.InteractionCreate, submissionID, voterID string, voteType vote.VoteType, reason string, replyToOriginal bool) {
	voteManager, err := vote.NewManager()
	if err != nil {
		log.Printf("Failed to create vote manager: %v", err)
		return
	}

	session, err := voteManager.LoadSession(submissionID)
	if err != nil {
		log.Printf("Failed to load vote session for submission %s: %v", submissionID, err)
		return
	}

	newVote := vote.Vote{
		VoterID:   voterID,
		Type:      voteType,
		Reason:    reason,
		Timestamp: time.Now(),
	}
	session.AddVote(newVote)

	if err := voteManager.SaveSession(session); err != nil {
		log.Printf("Failed to save vote session for submission %s: %v", submissionID, err)
		return
	}

	updateReviewMessage(s, i, session)
	processVoteResult(s, i, session, replyToOriginal)
}

// updateReviewMessage updates the review message with the current voting status.
func updateReviewMessage(s *discordgo.Session, i *discordgo.InteractionCreate, session *vote.Session) {
	originalEmbeds := i.Message.Embeds
	var voteSummary string
	for _, v := range session.Votes {
		if v.Type == vote.Reject && v.Reason != "" {
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
					Value: "前两票出现差异，等待第三票决定最终结果。",
				},
			}
		}
	}

	var updatedEmbeds []*discordgo.MessageEmbed
	existingVoteEmbedIndex := -1
	for idx, embed := range originalEmbeds {
		if embed.Title == "当前投票状态" {
			existingVoteEmbedIndex = idx
			break
		}
	}

	if existingVoteEmbedIndex != -1 {
		originalEmbeds[existingVoteEmbedIndex] = voteEmbed
		updatedEmbeds = originalEmbeds
	} else {
		updatedEmbeds = append(originalEmbeds, voteEmbed)
	}

	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &updatedEmbeds,
	})
	if err != nil {
		log.Printf("Failed to update review message for submission %s: %v", session.SubmissionID, err)
	}
}

// processVoteResult checks the votes and takes final action if needed.
func processVoteResult(s *discordgo.Session, i *discordgo.InteractionCreate, session *vote.Session, replyToOriginal bool) {
	if len(session.Votes) < 2 {
		return // Not enough votes to make a decision yet
	}

	voteCounts := make(map[vote.VoteType]int)
	for _, v := range session.Votes {
		voteCounts[v.Type]++
		// Feature vote also counts as a Pass vote
		if v.Type == vote.Feature {
			voteCounts[vote.Pass]++
		}
	}

	var finalStatus string
	var reviewerID string // For now, we'll just log the last voter.

	// Check for a two-vote consensus
	for voteType, count := range voteCounts {
		if count >= 2 {
			reviewerID = session.Votes[len(session.Votes)-1].VoterID
			switch voteType {
			case vote.Pass:
				// If we have 2 or more feature votes, the final status is "featured"
				if voteCounts[vote.Feature] >= 2 {
					finalStatus = "featured"
				} else {
					finalStatus = "approved"
				}
			case vote.Reject:
				finalStatus = "rejected"
			case vote.Ban:
				finalStatus = "banned" // We'll handle the actual ban action below
			}
			break // A decision has been reached
		}
	}

	// If there are 2 votes with no consensus, wait for a 3rd.
	if len(session.Votes) == 2 && !tools.HasConsensus(voteCounts) {
		return // Wait for the third vote
	}

	if len(session.Votes) >= 3 {
		// With 3 or more votes, the last vote is the tie-breaker.
		lastVoteType := session.Votes[len(session.Votes)-1].Type
		reviewerID = session.Votes[len(session.Votes)-1].VoterID

		switch lastVoteType {
		case vote.Pass:
			finalStatus = "approved"
		case vote.Feature:
			finalStatus = "featured"
		case vote.Reject:
			finalStatus = "rejected"
		case vote.Ban:
			finalStatus = "banned"
		}
	}

	if finalStatus == "" {
		return // No decision reached yet
	}

	submission, err := db.GetSubmission(session.SubmissionID)
	if err != nil {
		log.Printf("Could not get submission %s for final processing: %v", session.SubmissionID, err)
		return
	}

	oldStatus := submission.Status
	// Only proceed if the final status is different from the old status
	var rejectionReasons []string
	if finalStatus == "rejected" {
		for _, v := range session.Votes {
			if v.Type == vote.Reject && v.Reason != "" {
				rejectionReasons = append(rejectionReasons, v.Reason)
			}
		}
	}

	if finalStatus != oldStatus {
		handleStatusChange(s, submission, finalStatus, reviewerID, replyToOriginal)
	}

	finalizeReviewMessage(s, i, session.SubmissionID, finalStatus, rejectionReasons)
}

// handleStatusChange processes the consequences of a submission's final status.
func handleStatusChange(s *discordgo.Session, submission *model.Submission, finalStatus, reviewerID string, replyToOriginal bool) {
	// Update user stats based on the new status
	switch finalStatus {
	case "featured":
		db.IncrementFeaturedCount(submission.UserID)
	case "rejected":
		db.IncrementRejectedCount(submission.UserID)
	case "banned":
		db.BanUser(submission.UserID, "Banned through submission voting.")
		db.IncrementRejectedCount(submission.UserID) // Banned submissions are also considered rejected
		finalStatus = "rejected"                     // The submission status itself is 'rejected'
	}

	// Update submission status in the database
	if err := db.UpdateSubmissionReviewer(submission.ID, finalStatus, reviewerID); err != nil {
		log.Printf("Failed to update submission status for %s: %v", submission.ID, err)
		return
	}

	// If the submission was pending and is now approved or featured, send the publication messages.
	if submission.Status == "pending" && (finalStatus == "approved" || finalStatus == "featured") {
		publishMsg, err := sendPublicationMessage(s, submission)
		if err != nil {
			log.Printf("Error sending publication message for submission %s: %v", submission.ID, err)
		} else {
			if err := db.UpdateFinalAmwayMessageID(submission.ID, publishMsg.ID); err != nil {
				log.Printf("Error updating final amway message ID for submission %s: %v", submission.ID, err)
			}
			if replyToOriginal {
				sendNotificationToOriginalPost(s, submission, publishMsg)
			}
		}
	}
}

// finalizeReviewMessage updates the original review message to show the final result.
func finalizeReviewMessage(s *discordgo.Session, i *discordgo.InteractionCreate, submissionID, finalStatus string, reasons []string) {
	finalEmbed := &discordgo.MessageEmbed{
		Title:       "✅ 投票结束",
		Description: fmt.Sprintf("对投稿 `%s` 的投票已完成。\n\n**最终结果:** `%s`", submissionID, finalStatus),
		Color:       0x5865F2, // Discord Blurple
	}

	var components []discordgo.MessageComponent
	if finalStatus == "rejected" && len(reasons) > 0 {
		// Create a button for each reason, spread across multiple rows if necessary
		reasonButtons := []discordgo.MessageComponent{}
		for idx := range reasons {
			reasonButtons = append(reasonButtons, discordgo.Button{
				Label:    fmt.Sprintf("理由%d", idx+1),
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("select_reason:%s:%d", submissionID, idx),
			})
		}

		// Group buttons into ActionRows (max 5 per row)
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

		// Add the confirmation button in a new row
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "发送私信通知",
					Style:    discordgo.PrimaryButton,
					CustomID: "send_rejection_dm:" + submissionID,
				},
			},
		})
	}

	embeds := i.Message.Embeds
	embeds = append(embeds, finalEmbed)

	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &embeds,
		Components: &components,
	})
	if err != nil {
		log.Printf("Failed to finalize review message for submission %s: %v", submissionID, err)
	}
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
