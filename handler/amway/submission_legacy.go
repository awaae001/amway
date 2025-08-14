package amway

import (
	"amway/config"
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
