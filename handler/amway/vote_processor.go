package amway

import (
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
func processVote(s *discordgo.Session, i *discordgo.InteractionCreate, submissionID, voterID string, voteType vote.VoteType, reason string, replyToOriginal bool, cacheID string) {
	voteManager, err := vote.NewManager()
	if err != nil {
		log.Printf("Failed to create vote manager: %v", err)
		return
	}

	submission, err := db.GetSubmission(submissionID)
	if err != nil {
		log.Printf("Failed to get submission %s: %v", submissionID, err)
		return
	}
	if submission == nil {
		log.Printf("Submission %s not found", submissionID)
		return
	}

	session, err := voteManager.LoadSession(submission.VoteFileID)
	if err != nil {
		log.Printf("Failed to load vote session for submission %s (VoteFileID: %s): %v", submissionID, submission.VoteFileID, err)
		return
	}
	session.SubmissionID = submissionID // Keep the original submission ID for logic

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
	processVoteResult(s, i, session, replyToOriginal, cacheID)
}

// updateReviewMessage updates the review message with the current voting status.
func updateReviewMessage(s *discordgo.Session, i *discordgo.InteractionCreate, session *vote.Session) {
	voteEmbed := BuildVoteStatusEmbed(session)

	originalEmbeds := i.Message.Embeds
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
func processVoteResult(s *discordgo.Session, i *discordgo.InteractionCreate, session *vote.Session, replyToOriginal bool, cacheID string) {
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

	var banReason string
	if finalStatus == "banned" {
		for _, v := range session.Votes {
			if v.Type == vote.Ban && v.Reason != "" {
				banReason = v.Reason // Just get the last ban reason
				break
			}
		}
	}

	if finalStatus != oldStatus {
		handleStatusChange(s, submission, finalStatus, reviewerID, replyToOriginal, banReason)
	}

	finalizeReviewMessage(s, i, session.SubmissionID, finalStatus, rejectionReasons, cacheID)
}

// handleStatusChange processes the consequences of a submission's final status.
func handleStatusChange(s *discordgo.Session, submission *model.Submission, finalStatus, reviewerID string, replyToOriginal bool, banReason string) {
	// Update user stats based on the new status
	switch finalStatus {
	case "featured":
		db.IncrementFeaturedCount(submission.UserID)
	case "rejected":
		db.IncrementRejectedCount(submission.UserID)
	case "banned":
		// Apply a 3-day temporary ban and get the updated user stats.
		updatedUser, err := db.ApplyBan(submission.UserID, 3*24*time.Hour)
		if err != nil {
			log.Printf("Failed to apply temporary ban to user %s: %v", submission.UserID, err)
		} else {
			// Check if the user has reached the permanent ban threshold.
			if updatedUser.BanCount >= 3 {
				err := db.ApplyPermanentBan(submission.UserID)
				if err != nil {
					log.Printf("Failed to apply permanent ban to user %s: %v", submission.UserID, err)
				} else {
					// Notify the user about the permanent ban.
					sendBanNotification(s, submission.UserID, true, updatedUser.BanCount, banReason)
				}
			} else {
				// Notify the user about the temporary ban.
				sendBanNotification(s, submission.UserID, false, updatedUser.BanCount, banReason)
			}
		}

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
		PublishSubmission(s, submission, replyToOriginal)
	}
}

// finalizeReviewMessage updates the original review message to show the final result.
func finalizeReviewMessage(s *discordgo.Session, i *discordgo.InteractionCreate, submissionID, finalStatus string, reasons []string, cacheID string) {
	finalEmbed := BuildFinalVoteEmbed(submissionID, finalStatus)

	// If there are reasons, store them in the new cache so the selection handler can access them.
	if len(reasons) > 0 {
		model.SetAvailableRejectionReasons(submissionID, reasons)
	}

	components := BuildRejectionComponents(cacheID, reasons)

	embeds := i.Message.Embeds
	embeds = append(embeds, finalEmbed)

	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &embeds,
		Components: &components,
	})
	if err != nil {
		log.Printf("Failed to finalize review message for submission %s: %v", submissionID, err)
	}

	if finalStatus != "rejected" || len(reasons) == 0 {
		utils.RemoveFromCache(cacheID)
		model.DeleteAvailableRejectionReasons(submissionID) // Clean up the new cache as well
		log.Printf("Removed cache entry %s after voting completion for submission %s", cacheID, submissionID)
	}
}

// sendBanNotification sends a direct message to a user about their ban status.
func sendBanNotification(s *discordgo.Session, userID string, isPermanent bool, banCount int, reason string) {
	channel, err := s.UserChannelCreate(userID)
	if err != nil {
		log.Printf("Failed to create DM channel for user %s: %v", userID, err)
		return
	}

	var message string
	if isPermanent {
		message = fmt.Sprintf("您好，由于您在安利墙的行为，您的账户已被安利系统拒接投稿权限。这是您的第 %d 次封禁。\n\n**封禁理由**\n%s", banCount, reason)
	} else {
		message = fmt.Sprintf("您好，由于您在安利墙的行为，您的账户已被临时封禁3天。这是您的第 %d 次封禁。累计3次封禁将被永久拒绝投稿\n\n**封禁理由**\n%s", banCount, reason)
	}

	_, err = s.ChannelMessageSend(channel.ID, message)
	if err != nil {
		log.Printf("Failed to send ban notification to user %s: %v", userID, err)
	}
}
