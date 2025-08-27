package amway

import (
	"amway/vote"
	"fmt"
	"log"
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
	submissionID := parts[2]
	voterID := i.Member.User.ID

	if voteType == vote.Reject {
		// Show a modal for the rejection reason
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "modal_reject:" + submissionID,
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

	go processVote(s, i, submissionID, voterID, voteType, "")
}

// ModalRejectHandler handles the submission of the rejection reason modal.
func ModalRejectHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.Split(i.ModalSubmitData().CustomID, ":")
	if len(parts) != 2 {
		return // Invalid custom ID
	}
	submissionID := parts[1]
	voterID := i.Member.User.ID
	reason := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		fmt.Printf("Error sending deferred response: %v\n", err)
		return
	}

	go processVote(s, i, submissionID, voterID, vote.Reject, reason)
}
