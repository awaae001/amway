package amway

import (
	"amway/config"
	"amway/db"
	"amway/model"
	"amway/utils"
	"amway/vote"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// VoteHandler handles all voting interactions.
func VoteHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 1. Parse interaction data
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(parts) != 3 {
		return // Invalid custom ID format
	}
	voteType := vote.VoteType(parts[1])
	submissionID := parts[2]
	voterID := i.Member.User.ID

	// Immediately acknowledge the interaction to prevent timeout
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		fmt.Printf("Error sending deferred response: %v\n", err)
		return
	}

	go func() {
		// 2. Initialize Vote Manager
		voteManager, err := vote.NewManager()
		if err != nil {
			log.Printf("Failed to create vote manager: %v", err)
			return
		}

		// 3. Load or create a session
		session, err := voteManager.LoadSession(submissionID)
		if err != nil {
			log.Printf("Failed to load vote session for submission %s: %v", submissionID, err)
			return
		}

		// 4. Add the new vote
		newVote := vote.Vote{
			VoterID:   voterID,
			Type:      voteType,
			Timestamp: time.Now(),
		}
		session.AddVote(newVote)

		// 5. Save the session
		if err := voteManager.SaveSession(session); err != nil {
			log.Printf("Failed to save vote session for submission %s: %v", submissionID, err)
			return
		}

		// 6. Analyze votes and update message (implementation to follow)
		updateReviewMessage(s, i, session)

		// 7. Process the final result if conditions are met
		processVoteResult(s, i, session)
	}()
}

// updateReviewMessage updates the review message with the current voting status.
func updateReviewMessage(s *discordgo.Session, i *discordgo.InteractionCreate, session *vote.Session) {
	// TODO: Implement the logic to show who voted for what.
	// For now, just a simple confirmation.

	originalEmbeds := i.Message.Embeds

	var voteSummary string
	for _, v := range session.Votes {
		voteSummary += fmt.Sprintf("<@%s>投了 `%s`\n", v.VoterID, v.Type)
	}

	voteEmbed := &discordgo.MessageEmbed{
		Title:       "当前投票状态",
		Description: voteSummary,
		Color:       0x00BFFF, // Deep sky blue
	}

	updatedEmbeds := append(originalEmbeds, voteEmbed)

	// Check if this is the first vote. If so, create a new embed. If not, edit the existing one.
	var existingVoteEmbedIndex = -1
	for idx, embed := range originalEmbeds {
		if embed.Title == "当前投票状态" {
			existingVoteEmbedIndex = idx
			break
		}
	}

	if existingVoteEmbedIndex != -1 {
		// Update existing vote embed
		originalEmbeds[existingVoteEmbedIndex] = voteEmbed
		updatedEmbeds = originalEmbeds
	} else {
		// Append new vote embed
		updatedEmbeds = append(originalEmbeds, voteEmbed)
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &updatedEmbeds,
	})
}

// processVoteResult checks the votes and takes final action if needed.
func processVoteResult(s *discordgo.Session, i *discordgo.InteractionCreate, session *vote.Session) {
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

	if finalStatus == "" && len(session.Votes) == 2 {
		hasPass := voteCounts[vote.Pass] > 0
		hasReject := voteCounts[vote.Reject] > 0
		hasBan := voteCounts[vote.Ban] > 0

		if (hasPass && hasReject) || (hasPass && hasBan) {
			// Conflict, wait for a third vote
			return
		}
	}

	if finalStatus == "" {
		return // No consensus yet, or conflict with more than 2 voters
	}

	submission, err := db.GetSubmission(session.SubmissionID)
	if err != nil {
		log.Printf("Could not get submission %s for final processing: %v", session.SubmissionID, err)
		return
	}

	// Update user stats
	switch finalStatus {
	case "approved":
		// No change in stats for a simple approval
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
	if err := db.UpdateSubmissionReviewer(session.SubmissionID, finalStatus, reviewerID); err != nil {
		log.Printf("Failed to update submission status for %s: %v", session.SubmissionID, err)
		return
	}

	// If approved or featured, send to the publication channel
	if finalStatus == "approved" || finalStatus == "featured" {
		publishMsg, err := sendPublicationMessage(s, submission)
		if err != nil {
			log.Printf("Error sending publication message for submission %s: %v", session.SubmissionID, err)
		} else {
			if err := db.UpdateFinalAmwayMessageID(session.SubmissionID, publishMsg.ID); err != nil {
				log.Printf("Error updating final amway message ID for submission %s: %v", session.SubmissionID, err)
			}
			sendNotificationToOriginalPost(s, submission, publishMsg)
		}
	}

	// --- Finalize Review Message ---
	finalEmbed := &discordgo.MessageEmbed{
		Title:       "✅ 投票结束",
		Description: fmt.Sprintf("对投稿 `%s` 的投票已完成。\n\n**最终结果:** `%s`", session.SubmissionID, finalStatus),
		Color:       0x5865F2, // Discord Blurple
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{i.Message.Embeds[0], finalEmbed}, // Keep original submission info
		Components: &[]discordgo.MessageComponent{},                             // Remove buttons
	})
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
