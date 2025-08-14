package amway

import (
	"amway/model"
	"amway/utils"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// submissionModalHandler handles the submission modal form submission
func SubmissionModalHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	// Extract form data
	var url, title, content string
	for _, component := range data.Components {
		if actionRow, ok := component.(*discordgo.ActionsRow); ok {
			for _, comp := range actionRow.Components {
				if textInput, ok := comp.(*discordgo.TextInput); ok {
					switch textInput.CustomID {
					case "submission_url":
						url = textInput.Value
					case "submission_title":
						title = textInput.Value
					case "submission_content":
						content = textInput.Value
					}
				}
			}
		}
	}

	// Validate input
	if url == "" || title == "" || content == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "所有字段都是必填的，请重新提交。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Add submission to database
	submissionID, err := utils.AddSubmission(i.Member.User.ID, url, title, content, i.GuildID, i.Member.User.Username)
	if err != nil {
		fmt.Printf("Error adding submission to database: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "提交失败，请稍后再试。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Send confirmation to user
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "您的投稿已成功提交，正在等待审核。",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	// Use the new reusable function to send the review message
	submission := &model.Submission{
		ID:               submissionID,
		UserID:           i.Member.User.ID,
		OriginalTitle:    title,
		URL:              url,
		RecommendContent: content,
	}
	SendSubmissionToReviewChannel(s, submission)
}
