package amway

import (
	"amway/db"
	"amway/model"
	"amway/utils"
	"amway/vote"
	"fmt"
	"log"
	"strconv"
	"strings"

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
				Content: "投票请求已过期，请联系管理员",
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
				Content: "投票请求已过期，请联系管理员",
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
				Content: "投票请求已过期，请联系管理员",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	submissionID := cacheData.SubmissionID

	// Get the original reasons from the message component
	var allReasons []string
	for _, comp := range i.Message.Components {
		row, ok := comp.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, btn := range row.Components {
			button, ok := btn.(*discordgo.Button)
			if ok && strings.HasPrefix(button.CustomID, "select_reason:") {
				allReasons = append(allReasons, button.Label)
			}
		}
	}

	selectedReason := allReasons[reasonIndex]

	// Toggle selection in cache
	cachedReasons, _ := model.GetRejectionReasons(submissionID)
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
	model.SetRejectionReasons(submissionID, newReasons)

	adminActionUpdate(s, i)
}

// SendRejectionDMHandler handles sending the rejection DM to the user.
func SendRejectionDMHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cacheID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	// Get submission data from cache
	cacheData, found := utils.GetFromCache(cacheID)
	if !found {
		log.Printf("Cache data not found for cache ID: %s", cacheID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "投票请求已过期，请联系管理员",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	submissionID := cacheData.SubmissionID
	reasons, ok := model.GetRejectionReasons(submissionID)
	if !ok || len(reasons) == 0 {
		// Respond with an ephemeral message if no reasons were selected
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "请先选择至少一个“不通过”的理由。",
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
	model.DeleteRejectionReasons(submissionID)
	utils.RemoveFromCache(cacheID)
	log.Printf("Removed cache entry %s after sending rejection DM for submission %s", cacheID, submissionID)
	adminActionUpdate(s, i)
}

// adminActionUpdate updates the admin message, disabling components after action.
func adminActionUpdate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cacheID := strings.Split(i.MessageComponentData().CustomID, ":")[1]
	
	// Get submission data from cache (may fail if cache was already cleaned)
	cacheData, found := utils.GetFromCache(cacheID)
	var submissionID string
	if found {
		submissionID = cacheData.SubmissionID
	} else {
		// If cache is not found, we can't get the submissionID, but that's okay for DM sent scenario
		log.Printf("Cache already cleaned for cacheID %s", cacheID)
	}
	
	selectedReasons, _ := model.GetRejectionReasons(submissionID)

	// Create a map for quick lookup of selected reasons
	isSelected := make(map[string]bool)
	for _, r := range selectedReasons {
		isSelected[r] = true
	}

	// Disable all components if the DM has been sent
	dmSent := i.MessageComponentData().CustomID == "send_rejection_dm:"+cacheID

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

			// Clone the button to modify it
			newButton := *button
			if strings.HasPrefix(newButton.CustomID, "select_reason:") {
				if isSelected[newButton.Label] {
					newButton.Style = discordgo.SuccessButton // Green for selected
				} else {
					newButton.Style = discordgo.SecondaryButton // Gray for not selected
				}
			}
			if dmSent {
				newButton.Disabled = true
			}
			newRow.Components = append(newRow.Components, newButton)
		}
		newComponents = append(newComponents, newRow)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Components: newComponents,
		},
	})
}

func processVoteRemoval(s *discordgo.Session, i *discordgo.InteractionCreate, submissionID, voterID, cacheID string) {
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

	if removed := session.RemoveVote(voterID); removed {
		if err := voteManager.SaveSession(session); err != nil {
			log.Printf("Failed to save vote session for submission %s: %v", submissionID, err)
			return
		}
		updateReviewMessage(s, i, session)
		// Also re-evaluate the vote result after removal
		processVoteResult(s, i, session, false, cacheID) // Assuming replyToOriginal is false for this action
	}
	// If not removed, do nothing, the user might have clicked by mistake without voting.
	// The deferred update will just clear the "thinking" state.
}
