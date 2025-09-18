package amway

import (
	"amway/db"
	"amway/utils"
	"amway/vote"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// VoteHandler handles all voting interactions.
func VoteHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(parts) != 3 {
		return // Invalid custom ID format
	}
	voteType := vote.VoteType(parts[1])
	cacheID := parts[2] // This is now cacheID instead of submissionID
	voterID := i.Member.User.ID

	// Get submission data from cache
	cacheData, found := utils.GetFromCache(cacheID)
	if !found {
		log.Printf("Cache data not found for cache ID: %s", cacheID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "投票请求已过期，请联系开发者确认是否是 bot 重启导致的缓存丢失或者审核超时",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	submissionID := cacheData.SubmissionID

	switch voteType {
	case "remove":
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		})
		if err != nil {
			fmt.Printf("Error sending deferred response: %v\n", err)
			return
		}

		go processVoteRemoval(s, i, submissionID, voterID, cacheID)
		return
	case vote.Reject:
		// Show a modal for the rejection reason
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "modal_reject:" + cacheID,
				Title:    "输入不通过理由",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "reason",
								Label:       "不通过理由",
								Style:       discordgo.TextInputParagraph,
								Placeholder: "请输入不通过的理由...",
								Required:    true,
								MinLength:   8,
								MaxLength:   128,
							},
						},
					},
				},
			},
		})
		if err != nil {
			log.Printf("Error responding with modal: %v", err)
		}
		return // Stop processing, wait for modal submission
	case vote.Ban:
		// Show a modal for the ban reason
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "modal_ban:" + cacheID,
				Title:    "输入封禁理由",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "reason",
								Label:       "封禁理由",
								Style:       discordgo.TextInputParagraph,
								Placeholder: "请输入封禁用户的理由...",
								Required:    true,
								MinLength:   8,
								MaxLength:   128,
							},
						},
					},
				},
			},
		})
		if err != nil {
			log.Printf("Error responding with ban modal: %v", err)
		}
		return // Stop processing, wait for modal submission
	}

	// For other vote types, defer the update and process in the background.
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		fmt.Printf("Error sending deferred response: %v\n", err)
		return
	}

	go processVote(s, i, submissionID, voterID, voteType, "", cacheData.ReplyToOriginal, cacheID)
}

// ModalRejectHandler handles the submission of the rejection reason modal.
func ModalRejectHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.Split(i.ModalSubmitData().CustomID, ":")
	if len(parts) != 2 {
		return // Invalid custom ID
	}
	cacheID := parts[1]
	voterID := i.Member.User.ID
	reason := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	// Get submission data from cache
	cacheData, found := utils.GetFromCache(cacheID)
	if !found {
		log.Printf("Cache data not found for cache ID: %s", cacheID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "投票请求已过期，请联系开发者确认是否是 bot 重启导致的缓存丢失或者审核超时",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	submissionID := cacheData.SubmissionID

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		fmt.Printf("Error sending deferred response: %v\n", err)
		return
	}

	go processVote(s, i, submissionID, voterID, vote.Reject, reason, cacheData.ReplyToOriginal, cacheID)
}

// SelectReasonHandler handles the selection of rejection reasons via buttons.
func SelectReasonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	cacheID := parts[1]
	reasonIndex, _ := strconv.Atoi(parts[2])

	// Get submission data from cache
	cacheData, found := utils.GetFromCache(cacheID)
	if !found {
		log.Printf("Cache data not found for cache ID: %s", cacheID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "投票请求已过期，请联系开发者确认是否是 bot 重启导致的缓存丢失或者审核超时",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	submissionID := cacheData.SubmissionID

	// Get all available reasons from the new cache
	allReasons, ok := utils.GetAvailableRejectionReasons(submissionID)
	if !ok || len(allReasons) <= reasonIndex {
		log.Printf("Available rejection reasons not found or index out of bounds for submission %s", submissionID)
		return
	}

	selectedReason := allReasons[reasonIndex]

	// Toggle selection in cache
	cachedReasons, _ := utils.GetRejectionReasons(submissionID)
	var newReasons []string
	reasonFound := false
	for _, r := range cachedReasons {
		if r == selectedReason {
			reasonFound = true
		} else {
			newReasons = append(newReasons, r)
		}
	}
	if !reasonFound {
		newReasons = append(newReasons, selectedReason)
	}
	utils.SetRejectionReasons(submissionID, newReasons)

	adminActionUpdate(s, i, false)
}

// SendRejectionDMHandler handles sending the rejection DM to the user.
func SendRejectionDMHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		log.Printf("Failed to defer interaction: %v", err)
		return
	}

	cacheID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	// Get submission data from cache
	cacheData, found := utils.GetFromCache(cacheID)
	if !found {
		log.Printf("Cache data not found for cache ID: %s", cacheID)
		handleExpiredInteraction(s, i)
		return
	}

	submissionID := cacheData.SubmissionID
	reasons, ok := utils.GetRejectionReasons(submissionID)
	if !ok || len(reasons) == 0 {
		// Respond with an ephemeral message if no reasons were selected
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "请先选择至少一个“不通过”的理由",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	submission, err := db.GetSubmission(submissionID)
	if err != nil {
		log.Printf("Could not get submission %s for DM: %v", submissionID, err)
		return
	}

	// Create and send the DM
	dmEmbed := &discordgo.MessageEmbed{
		Title:       "您的投稿未通过审核",
		Description: "很遗憾，您提交的以下安利投稿未通过审核：",
		Color:       0xFF0000, // Red
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "您的安利标题",
				Value: submission.RecommendTitle,
			},
			{
				Name:  "不通过理由",
				Value: "- " + strings.Join(reasons, "\n- "),
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "感谢您的参与，期待您下次的分享！",
		},
	}

	userChannel, err := s.UserChannelCreate(submission.UserID)
	if err != nil {
		log.Printf("Could not create DM channel for user %s: %v", submission.UserID, err)
		return
	}
	// Send the embed first
	_, err = s.ChannelMessageSendEmbed(userChannel.ID, dmEmbed)
	if err != nil {
		log.Printf("Could not send DM embed to user %s: %v", submission.UserID, err)
		// We can still try to send the plain text content
	}
	// Send the plain content for easy copying
	_, err = s.ChannelMessageSend(userChannel.ID, fmt.Sprintf("```\n%s\n```", submission.RecommendContent))
	if err != nil {
		log.Printf("Could not send DM plain text to user %s: %v", submission.UserID, err)
	}

	// Cleanup cache and update the original message
	utils.DeleteRejectionReasons(submissionID)
	utils.DeleteAvailableRejectionReasons(submissionID) // Clean up the new cache as well
	utils.RemoveFromCache(cacheID)
	log.Printf("Removed cache entry %s after sending rejection DM for submission %s", cacheID, submissionID)
	adminActionUpdate(s, i, true)
}

// adminActionUpdate updates the admin message, disabling components after action.
func adminActionUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, useEdit bool) {
	customIDParts := strings.Split(i.MessageComponentData().CustomID, ":")
	action := customIDParts[0]
	cacheID := customIDParts[1]

	cacheData, found := utils.GetFromCache(cacheID)
	var submissionID string
	if found {
		submissionID = cacheData.SubmissionID
	} else {
		log.Printf("Cache already cleaned for cacheID %s", cacheID)
	}

	selectedRejectionReasons, _ := utils.GetRejectionReasons(submissionID)
	selectedBanReasons, _ := utils.GetBanReasons(submissionID)

	isRejectionSelected := make(map[string]bool)
	for _, r := range selectedRejectionReasons {
		isRejectionSelected[r] = true
	}
	isBanSelected := make(map[string]bool)
	for _, r := range selectedBanReasons {
		isBanSelected[r] = true
	}

	dmSent := action == "send_rejection_dm" || action == "send_ban_dm"

	newComponents := []discordgo.MessageComponent{}
	for _, comp := range i.Message.Components {
		row, ok := comp.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		newRow := discordgo.ActionsRow{Components: []discordgo.MessageComponent{}}
		for _, btn := range row.Components {
			button, ok := btn.(*discordgo.Button)
			if !ok {
				continue
			}

			newButton := *button
			if strings.HasPrefix(newButton.CustomID, "select_reason:") {
				if isRejectionSelected[newButton.Label] {
					newButton.Style = discordgo.SuccessButton
				} else {
					newButton.Style = discordgo.SecondaryButton
				}
			} else if strings.HasPrefix(newButton.CustomID, "select_ban_reason:") {
				if isBanSelected[newButton.Label] {
					newButton.Style = discordgo.SuccessButton
				} else {
					newButton.Style = discordgo.SecondaryButton
				}
			}

			if dmSent {
				newButton.Disabled = true
			}
			newRow.Components = append(newRow.Components, newButton)
		}
		newComponents = append(newComponents, newRow)
	}

	if useEdit {
		content := i.Message.Content
		embeds := i.Message.Embeds
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content:    &content,
			Embeds:     &embeds,
			Components: &newComponents,
		})
		if err != nil {
			log.Printf("Failed to edit interaction response: %v", err)
		}
	} else {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    i.Message.Content,
				Embeds:     i.Message.Embeds,
				Components: newComponents,
			},
		})
		if err != nil {
			log.Printf("Failed to respond to interaction: %v", err)
		}
	}
}

func processVoteRemoval(s *discordgo.Session, i *discordgo.InteractionCreate, submissionID, voterID, cacheID string) {
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

	if removed := session.RemoveVote(voterID); removed {
		if err := voteManager.SaveSession(session); err != nil {
			log.Printf("Failed to save vote session for submission %s: %v", submissionID, err)
			return
		}
		updateReviewMessage(s, i, session)
		// Also re-evaluate the vote result after removal
		processVoteResult(s, i, session, false, cacheID) // Assuming replyToOriginal is false for this action
	}
}

// ModalBanHandler handles the submission of the ban reason modal.
func ModalBanHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.Split(i.ModalSubmitData().CustomID, ":")
	if len(parts) != 2 {
		return // Invalid custom ID
	}
	cacheID := parts[1]
	voterID := i.Member.User.ID
	reason := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	// Get submission data from cache
	cacheData, found := utils.GetFromCache(cacheID)
	if !found {
		log.Printf("Cache data not found for cache ID: %s", cacheID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "投票请求已过期，请联系开发者确认是否是 bot 重启导致的缓存丢失或者审核超时",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	submissionID := cacheData.SubmissionID

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		fmt.Printf("Error sending deferred response: %v\n", err)
		return
	}

	go processVote(s, i, submissionID, voterID, vote.Ban, reason, cacheData.ReplyToOriginal, cacheID)
}

// SelectBanReasonHandler handles the selection of ban reasons via buttons.
func SelectBanReasonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	cacheID := parts[1]
	reasonIndex, _ := strconv.Atoi(parts[2])

	cacheData, found := utils.GetFromCache(cacheID)
	if !found {
		handleExpiredInteraction(s, i)
		return
	}
	submissionID := cacheData.SubmissionID

	allReasons, ok := utils.GetAvailableBanReasons(submissionID)
	if !ok || len(allReasons) <= reasonIndex {
		return
	}
	selectedReason := allReasons[reasonIndex]

	cachedReasons, _ := utils.GetBanReasons(submissionID)
	var newReasons []string
	reasonFound := false
	for _, r := range cachedReasons {
		if r == selectedReason {
			reasonFound = true
			break
		}
	}
	if !reasonFound {
		// For ban reasons, we only allow one to be selected.
		newReasons = []string{selectedReason}
	} else {
		// Deselect
		newReasons = []string{}
	}

	utils.SetBanReasons(submissionID, newReasons)
	adminActionUpdate(s, i, false)
}

// SendBanDMHandler handles sending the ban DM to the user.
func SendBanDMHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		log.Printf("Failed to defer interaction: %v", err)
		return
	}

	cacheID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	cacheData, found := utils.GetFromCache(cacheID)
	if !found {
		// Handle expired interaction
		return
	}
	submissionID := cacheData.SubmissionID
	reasons, ok := utils.GetBanReasons(submissionID)
	if !ok || len(reasons) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "请先选择一个“封禁”的理由",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	selectedReason := reasons[0] // We only allow one reason for ban

	submission, err := db.GetSubmission(submissionID)
	if err != nil {
		log.Printf("Could not get submission %s for ban DM: %v", submissionID, err)
		return
	}

	// This logic is moved from handleStatusChange
	updatedUser, err := db.ApplyBan(submission.UserID, 3*24*time.Hour)
	if err != nil {
		log.Printf("Failed to apply temporary ban to user %s: %v", submission.UserID, err)
		return
	}

	isPermanent := false
	if updatedUser.BanCount >= 3 {
		err := db.ApplyPermanentBan(submission.UserID)
		if err != nil {
			log.Printf("Failed to apply permanent ban to user %s: %v", submission.UserID, err)
		} else {
			isPermanent = true
		}
	}

	sendBanNotification(s, submission.UserID, isPermanent, updatedUser.BanCount, selectedReason)

	// Cleanup cache and update the original message
	utils.DeleteBanReasons(submissionID)
	utils.DeleteAvailableBanReasons(submissionID)
	utils.RemoveFromCache(cacheID)
	log.Printf("Removed cache entry %s after sending ban DM for submission %s", cacheID, submissionID)
	adminActionUpdate(s, i, true)
}

func handleExpiredInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Disable all components on the original message
	emptyComponents := []discordgo.MessageComponent{}
	content := i.Message.Content
	embeds := i.Message.Embeds
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:    &content,
		Embeds:     &embeds,
		Components: &emptyComponents,
	})
	if err != nil {
		log.Printf("Failed to edit interaction response for expired interaction: %v", err)
	}

	// Send a followup message to inform the user
	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "该操作已过期，可能是由于机器人重启或审核超时消息按钮已移除",
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}
