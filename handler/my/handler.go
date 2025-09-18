package my

import (
	"amway/config"
	"amway/db"
	"amway/handler/amway"
	"amway/utils"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// MyAmwayButtonHandler handles the initial click on the "My Amway" button.
// It fetches the user's profile and the first page of their submissions.
func MyAmwayButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := i.Member.User
	page := 1

	// Fetch the first page of submissions (3 items)
	submissions, total, err := db.MyAmwayGetUserSubmissions(user.ID, page, PageSize)
	if err != nil {
		log.Printf("Error getting user submissions: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ 查询您的投稿记录时出错",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	responseData, err := BuildMyAmwayPanelComponents(user, submissions, page, total)
	if err != nil {
		log.Printf("Error building my amway panel: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ 构建您的个人面板时出错",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: responseData,
	})
}

// MyAmwayPageHandler handles the pagination button clicks.
func MyAmwayPageHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	userID, page, err := ParseMyAmwayPageCustomID(customID)
	if err != nil {
		log.Printf("Error parsing page custom ID: %v", err)
		return
	}

	// Permission check
	if i.Member.User.ID != userID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ 您不能操作不属于您的面板",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Fetch the submissions for the requested page
	submissions, total, err := db.MyAmwayGetUserSubmissions(userID, page, PageSize)
	if err != nil {
		log.Printf("Error getting user submissions for page %d: %v", page, err)
		// Handle error
		return
	}

	responseData, err := BuildMyAmwayPanelComponents(i.Member.User, submissions, page, total)
	if err != nil {
		log.Printf("Error building my amway panel for page %d: %v", page, err)
		// Handle error
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: responseData,
	})
}

// ModifyAmwayButtonHandler handles the click on the "Modify Amway" button.
func ModifyAmwayButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(parts) != 2 {
		return // Invalid custom id
	}
	userID := parts[1]

	// Permission check
	if i.Member.User.ID != userID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ 您不能操作不属于您的面板",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	modal := BuildModifyAmwayModal(userID)
	s.InteractionRespond(i.Interaction, modal)
}

// ModifyAmwayModalHandler handles the submission of the modification modal.
func ModifyAmwayModalHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	submissionID := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	userID := i.Member.User.ID

	submission, err := db.GetSubmission(submissionID)
	if err != nil || submission == nil {
		log.Printf("Error getting submission %s: %v", submissionID, err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ 未找到ID为 `%s` 的投稿", submissionID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Permission check
	if submission.UserID != userID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ 您无权修改此投稿",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Build and show the modification panel
	panel := BuildModificationPanel(submission)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: panel,
	})
}

// RetractPostHandler handles retracting the message from the original post's thread.
func RetractPostHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.TrimPrefix(i.MessageComponentData().CustomID, "retract_post_button:")
	userID := i.Member.User.ID

	// Get submission to get message IDs and URL
	submission, err := db.GetSubmission(submissionID)
	if err != nil || submission == nil {
		// Handle error
		return
	}

	// Permission check
	if submission.UserID != userID {
		// Handle error
		return
	}

	// Delete the message in the thread
	if submission.ThreadMessageID != "0" && submission.ThreadMessageID != "" {
		if originalChannelID, _, err := utils.GetOriginalPostDetails(submission.URL); err == nil {
			if err := s.ChannelMessageDelete(originalChannelID, submission.ThreadMessageID); err != nil {
				log.Printf("Failed to delete forwarded message %s in channel %s: %v", submission.ThreadMessageID, originalChannelID, err)
			}
		}
	}

	// Update the database record
	updatedSubmission, err := db.RetractAmwayPost(submissionID, userID)
	if err != nil {
		log.Printf("Error retracting post for submission %s: %v", submissionID, err)
		// Respond with error
		return
	}

	// Update the interaction message with the new panel state
	panel := BuildModificationPanel(updatedSubmission)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: panel,
	})
}

// ToggleAnonymityHandler handles toggling the anonymity of a submission.
func ToggleAnonymityHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.TrimPrefix(i.MessageComponentData().CustomID, "toggle_anonymity_button:")
	userID := i.Member.User.ID

	// 1. Toggle anonymity in the database
	err := db.ToggleAnonymity(submissionID, userID)
	if err != nil {
		log.Printf("Error toggling anonymity for submission %s: %v", submissionID, err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("❌ 切换匿名状态失败: %v", err)},
		})
		return
	}

	// 2. Fetch the updated submission
	updatedSubmission, err := db.GetSubmission(submissionID)
	if err != nil || updatedSubmission == nil {
		log.Printf("Error fetching updated submission %s: %v", submissionID, err)
		// Even if we can't update the message, we should update the panel
		return
	}

	// 3. Edit the original publication message
	if updatedSubmission.FinalAmwayMessageID != "" {
		publicationMessage, err := amway.BuildPublicationMessage(updatedSubmission)
		if err != nil {
			log.Printf("Error building publication message for submission %s: %v", updatedSubmission.ID, err)
		} else {
			_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Channel: config.Cfg.AmwayBot.Amway.PublishChannelID,
				ID:      updatedSubmission.FinalAmwayMessageID,
				Content: &publicationMessage.Content,
				Embeds:  &[]*discordgo.MessageEmbed{publicationMessage.Embed},
			})
			if err != nil {
				log.Printf("Error editing publication message for submission %s: %v", updatedSubmission.ID, err)
			}
		}
	}

	// 4. Update the interaction message with the new panel state
	panel := BuildModificationPanel(updatedSubmission)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: panel,
	})
}

// DeleteAmwayHandler handles the permanent deletion of a submission.
func DeleteAmwayHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.TrimPrefix(i.MessageComponentData().CustomID, "delete_amway_button:")
	userID := i.Member.User.ID

	// Get submission details before deleting
	submission, err := db.GetSubmission(submissionID)
	if err != nil || submission == nil {
		// Handle error
		return
	}

	// Permission check
	if submission.UserID != userID {
		// Handle error
		return
	}

	// Delete messages from Discord
	if submission.FinalAmwayMessageID != "" {
		amwayChannelID := config.Cfg.AmwayBot.Amway.PublishChannelID
		if err := s.ChannelMessageDelete(amwayChannelID, submission.FinalAmwayMessageID); err != nil {
			log.Printf("Failed to delete amway message %s in channel %s: %v", submission.FinalAmwayMessageID, amwayChannelID, err)
		}
	}
	if submission.ThreadMessageID != "0" && submission.ThreadMessageID != "" {
		if originalChannelID, _, err := utils.GetOriginalPostDetails(submission.URL); err == nil {
			if err := s.ChannelMessageDelete(originalChannelID, submission.ThreadMessageID); err != nil {
				log.Printf("Failed to delete forwarded message %s in channel %s: %v", submission.ThreadMessageID, originalChannelID, err)
			}
		}
	}

	// Perform hard delete from the database
	if err := db.DeleteSubmission(submissionID); err != nil {
		log.Printf("Error deleting submission %s: %v", submissionID, err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ 删除投稿 `%s` 时发生数据库错误", submissionID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Respond with a success message, replacing the modification panel
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf("✅ 投稿 `%s` 已被永久删除", submissionID),
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{},
		},
	})
}

// BackToMyAmwayHandler handles the back button click to return to the main My Amway panel.
func BackToMyAmwayHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(parts) != 2 {
		return // Invalid custom id
	}
	userID := parts[1]

	// Permission check
	if i.Member.User.ID != userID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ 您不能操作不属于您的面板",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Default to page 1 when returning
	page := 1

	// Fetch the first page of submissions (3 items)
	submissions, total, err := db.MyAmwayGetUserSubmissions(userID, page, PageSize)
	if err != nil {
		log.Printf("Error getting user submissions: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ 查询您的投稿记录时出错",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	responseData, err := BuildMyAmwayPanelComponents(i.Member.User, submissions, page, total)
	if err != nil {
		log.Printf("Error building my amway panel: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ 构建您的个人面板时出错",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: responseData,
	})
}
