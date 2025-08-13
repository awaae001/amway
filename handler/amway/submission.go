package amway

import (
	"amway/config"
	"amway/handler"
	"amway/utils"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func init() {
	handler.AddComponentHandler("create_submission_button", createSubmissionButtonHandler)
	handler.AddModalHandler("submission_modal", submissionModalHandler)
	handler.AddComponentHandler("approve_submission", approveSubmissionHandler)
	handler.AddComponentHandler("reject_submission", rejectSubmissionHandler)
	handler.AddComponentHandler("ignore_submission", ignoreSubmissionHandler)
	handler.AddComponentHandler("ban_submission", banSubmissionHandler)
	handler.AddComponentHandler("delete_submission", deleteSubmissionHandler)
}

func createSubmissionButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 检查用户是否被封禁
	banned, err := utils.IsUserBanned(i.Member.User.ID)
	if err != nil {
		fmt.Printf("Error checking if user is banned: %v\n", err)
		// 即使检查出错，也向用户显示一个通用错误，避免泄露内部问题
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "无法处理您的请求，请稍后再试。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if banned {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "您已被禁止投稿。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// 如果用户未被封禁，弹出 Modal
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "submission_modal",
			Title:    "创建新的投稿",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "submission_url",
							Label:       "URL",
							Style:       discordgo.TextInputShort,
							Placeholder: "请输入有效的URL",
							Required:    true,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "submission_title",
							Label:       "标题",
							Style:       discordgo.TextInputShort,
							Placeholder: "请输入简评的标题",
							Required:    true,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "submission_content",
							Label:       "内容",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "请输入您的简评内容",
							Required:    true,
						},
					},
				},
			},
		},
	})

	if err != nil {
		fmt.Printf("Error creating modal: %v\n", err)
	}
}

// submissionModalHandler handles the submission modal form submission
func submissionModalHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	submissionID, err := utils.AddSubmission(i.Member.User.ID, url, title, content)
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

	// Send review message to review channel
	reviewChannelID := config.Cfg.AmwayBot.Amway.ReviewChannelID
	if reviewChannelID == "" {
		fmt.Printf("Review channel ID not configured\n")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "新的投稿待审核",
		Description: fmt.Sprintf("**投稿ID:** %s\n**作者:** <@%s>\n**标题:** %s\n**URL:** %s\n**内容:**\n%s", submissionID, i.Member.User.ID, title, url, content),
		Color:       0xFFFF00, // Yellow color for pending
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("提交时间 • ID: %s", submissionID),
		},
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "通过",
					Style:    discordgo.SuccessButton,
					CustomID: "approve_submission:" + submissionID,
					Emoji:    &discordgo.ComponentEmoji{Name: "✅"},
				},
				discordgo.Button{
					Label:    "拒绝",
					Style:    discordgo.DangerButton,
					CustomID: "reject_submission:" + submissionID,
					Emoji:    &discordgo.ComponentEmoji{Name: "❌"},
				},
				discordgo.Button{
					Label:    "忽略",
					Style:    discordgo.SecondaryButton,
					CustomID: "ignore_submission:" + submissionID,
					Emoji:    &discordgo.ComponentEmoji{Name: "⏭️"},
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "封禁",
					Style:    discordgo.DangerButton,
					CustomID: "ban_submission:" + submissionID,
					Emoji:    &discordgo.ComponentEmoji{Name: "🔨"},
				},
				discordgo.Button{
					Label:    "删除",
					Style:    discordgo.DangerButton,
					CustomID: "delete_submission:" + submissionID,
					Emoji:    &discordgo.ComponentEmoji{Name: "🗑️"},
				},
			},
		},
	}

	_, err = s.ChannelMessageSendComplex(reviewChannelID, &discordgo.MessageSend{
		Embed:      embed,
		Components: components,
	})

	if err != nil {
		fmt.Printf("Error sending review message: %v\n", err)
	}
}

// approveSubmissionHandler handles approval of submissions
func approveSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	// Get submission details
	submission, err := utils.GetSubmission(submissionID)
	if err != nil {
		fmt.Printf("Error getting submission: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "无法找到该投稿。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update submission status
	err = utils.UpdateSubmissionStatus(submissionID, "approved")
	if err != nil {
		fmt.Printf("Error updating submission status: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "更新状态失败。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Send to publish channel
	publishChannelID := config.Cfg.AmwayBot.Amway.PublishChannelID
	if publishChannelID != "" {
		embed := &discordgo.MessageEmbed{
			Description: submission.Content,
			Color:       0x00FF00, // Green color for approved
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("作者: %s", submission.UserID),
			},
		}

		if submission.URL != "" {
			embed.URL = submission.URL
		}

		publishMsg, err := s.ChannelMessageSendEmbed(publishChannelID, embed)
		if err != nil {
			fmt.Printf("Error sending to publish channel: %v\n", err)
		} else {
			// Add reactions
			s.MessageReactionAdd(publishChannelID, publishMsg.ID, "👍")
			s.MessageReactionAdd(publishChannelID, publishMsg.ID, "❓")
			s.MessageReactionAdd(publishChannelID, publishMsg.ID, "👎")
		}
	}

	// Update the review message
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "投稿已通过",
					Description: fmt.Sprintf("**投稿ID:** %s\n**审核员:** <@%s>\n**状态:** ✅ 已通过并发布", submissionID, i.Member.User.ID),
					Color:       0x00FF00,
				},
			},
			Components: []discordgo.MessageComponent{}, // Remove buttons
		},
	})
}

// rejectSubmissionHandler handles rejection of submissions
func rejectSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	err := utils.UpdateSubmissionStatus(submissionID, "rejected")
	if err != nil {
		fmt.Printf("Error updating submission status: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "更新状态失败。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "投稿已拒绝",
					Description: fmt.Sprintf("**投稿ID:** %s\n**审核员:** <@%s>\n**状态:** ❌ 已拒绝", submissionID, i.Member.User.ID),
					Color:       0xFF0000,
				},
			},
			Components: []discordgo.MessageComponent{},
		},
	})
}

// ignoreSubmissionHandler handles ignoring submissions
func ignoreSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	err := utils.UpdateSubmissionStatus(submissionID, "ignored")
	if err != nil {
		fmt.Printf("Error updating submission status: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "更新状态失败。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "投稿已忽略",
					Description: fmt.Sprintf("**投稿ID:** %s\n**审核员:** <@%s>\n**状态:** ⏭️ 已忽略", submissionID, i.Member.User.ID),
					Color:       0x808080,
				},
			},
			Components: []discordgo.MessageComponent{},
		},
	})
}

// banSubmissionHandler handles banning users and their submissions
func banSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	// Get submission to get user ID
	submission, err := utils.GetSubmission(submissionID)
	if err != nil {
		fmt.Printf("Error getting submission: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "无法找到该投稿。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Ban the user
	err = utils.BanUser(submission.UserID, "违规投稿")
	if err != nil {
		fmt.Printf("Error banning user: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "封禁用户失败。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update submission status
	err = utils.UpdateSubmissionStatus(submissionID, "rejected")
	if err != nil {
		fmt.Printf("Error updating submission status: %v\n", err)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "用户已封禁",
					Description: fmt.Sprintf("**投稿ID:** %s\n**审核员:** <@%s>\n**状态:** 🔨 用户已封禁，投稿已拒绝", submissionID, i.Member.User.ID),
					Color:       0x8B0000,
				},
			},
			Components: []discordgo.MessageComponent{},
		},
	})
}

// deleteSubmissionHandler handles deletion of submissions
func deleteSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	submissionID := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	err := utils.DeleteSubmission(submissionID)
	if err != nil {
		fmt.Printf("Error deleting submission: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "删除投稿失败。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "投稿已删除",
					Description: fmt.Sprintf("**投稿ID:** %s\n**审核员:** <@%s>\n**状态:** 🗑️ 已删除", submissionID, i.Member.User.ID),
					Color:       0x000000,
				},
			},
			Components: []discordgo.MessageComponent{},
		},
	})
}
