package my

import (
	"amway/config"
	"amway/db"
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
				Content: "❌ 查询您的投稿记录时出错。",
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
				Content: "❌ 构建您的个人面板时出错。",
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
				Content: "❌ 您不能操作不属于您的面板。",
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

// RetractSubmissionButtonHandler handles the click on the "Retract Submission" button.
func RetractSubmissionButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
				Content: "❌ 您不能操作不属于您的面板。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	modal := BuildRetractModal(userID)
	s.InteractionRespond(i.Interaction, modal)
}

// RetractSubmissionModalHandler handles the submission of the retraction modal.
func RetractSubmissionModalHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	submissionID := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	userID := i.Member.User.ID

	submission, err := db.MyAmwayRetractSubmission(submissionID, userID)
	if err != nil {
		log.Printf("Error retracting submission %s for user %s: %v", submissionID, userID, err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ 撤回失败: %s", err.Error()),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Now, delete the messages from the Discord channels
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

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ 投稿 `%s` 已成功撤回！", submissionID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	// Optionally, you can update the original panel message here to reflect the change.
}
