package amway

import (
	"amway/config"
	"amway/utils"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func CreateSubmissionButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	// 如果用户未被封禁，弹出链接验证模态框
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "submission_link_modal",
			Title:    "投稿第一步：帖子链接",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "submission_url",
							Label:       "Discord帖子链接",
							Style:       discordgo.TextInputShort,
							Placeholder: "请输入Discord帖子或频道链接",
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

// linkSubmissionHandler handles the link validation modal submission
func LinkSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	// Extract URL from form
	var url string
	for _, component := range data.Components {
		if actionRow, ok := component.(*discordgo.ActionsRow); ok {
			for _, comp := range actionRow.Components {
				if textInput, ok := comp.(*discordgo.TextInput); ok {
					if textInput.CustomID == "submission_url" {
						url = textInput.Value
						break
					}
				}
			}
		}
	}

	if url == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "请提供有效的链接。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// 验证Discord帖子链接
	currentGuildID := i.GuildID
	postInfo, err := utils.ValidateDiscordPost(s, url, currentGuildID, i.Member.User.ID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("验证失败：%v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// 显示帖子信息并要求确认
	postInfoText := utils.FormatDiscordPostInfo(postInfo)

	embed := &discordgo.MessageEmbed{
		Title:       "帖子信息确认",
		Description: fmt.Sprintf("%s\n\n请确认以上信息无误，然后点击下方按钮继续填写安利内容。", postInfoText),
		Color:       0x00FF00,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "确认并继续",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("confirm_post:%s:%s:%s", postInfo.ChannelID, postInfo.MessageID, postInfo.Author.ID),
					Emoji:    &discordgo.ComponentEmoji{Name: "✅"},
				},
				discordgo.Button{
					Label:    "取消",
					Style:    discordgo.DangerButton,
					CustomID: "cancel_submission",
					Emoji:    &discordgo.ComponentEmoji{Name: "❌"},
				},
			},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

// confirmPostHandler handles the post confirmation button click
func ConfirmPostHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "数据格式错误，请重新开始投稿流程。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// 弹出第二步内容填写模态框
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("submission_content_modal:%s", strings.Join(parts[1:], ":")),
			Title:    "投稿第二步：安利内容",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "recommend_title",
							Label:       "安利标题",
							Style:       discordgo.TextInputShort,
							Placeholder: "请输入您的安利标题",
							Required:    true,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "recommend_content",
							Label:       "安利内容",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "请输入您的安利内容和推荐理由",
							Required:    true,
						},
					},
				},
			},
		},
	})

	if err != nil {
		fmt.Printf("Error creating content modal: %v\n", err)
	}
}

// cancelSubmissionHandler handles the cancel button click
func CancelSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "投稿已取消。",
			Components: []discordgo.MessageComponent{},
			Embeds:     []*discordgo.MessageEmbed{},
		},
	})
}

// contentSubmissionHandler handles the final content submission
func ContentSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	// Extract post info from custom ID
	customID := data.CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "数据格式错误，请重新开始投稿流程。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	guildID := i.GuildID // 从交互中获取当前服务器ID
	channelID := parts[1]
	messageID := parts[2]
	originalAuthor := parts[3]

	// 构造完整的原帖链接
	originalURL := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", guildID, channelID, messageID)

	// 为了获取时间戳和消息数量，重新验证一次帖子
	postInfo, err := utils.ValidateDiscordPost(s, originalURL, guildID, i.Member.User.ID)
	if err != nil {
		// 如果验证失败，打印错误但继续流程，只是没有时间戳和消息数量
		fmt.Printf("二次验证帖子以获取元数据时出错: %v\n", err)
	}

	var originalPostTimestamp string
	if postInfo != nil {
		originalPostTimestamp = postInfo.Timestamp
	}

	// Extract form data
	var recommendTitle, recommendContent string
	for _, component := range data.Components {
		if actionRow, ok := component.(*discordgo.ActionsRow); ok {
			for _, comp := range actionRow.Components {
				if textInput, ok := comp.(*discordgo.TextInput); ok {
					switch textInput.CustomID {
					case "recommend_title":
						recommendTitle = textInput.Value
					case "recommend_content":
						recommendContent = textInput.Value
					}
				}
			}
		}
	}

	if recommendTitle == "" || recommendContent == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "标题和内容都是必填的，请重新提交。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Add submission to database using new V2 function
	submissionID, err := utils.AddSubmissionV2(
		i.Member.User.ID, originalURL,
		recommendTitle, recommendContent,
		"", originalAuthor, originalPostTimestamp, i.GuildID, i.Member.User.Username)
	if err != nil {
		fmt.Printf("Error adding submission to database: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("提交失败，请稍后再试。错误详情: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	// Send confirmation to user
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "您的安利投稿已成功提交，正在等待审核。",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	// Send review message to review channel (需要更新审核消息格式)
	reviewChannelID := config.Cfg.AmwayBot.Amway.ReviewChannelID
	if reviewChannelID == "" {
		fmt.Printf("Review channel ID not configured\n")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "新的安利投稿待审核",
		Description: fmt.Sprintf("**投稿ID:** %s\n**投稿人:** <@%s>\n**安利标题:** %s\n**原帖作者:** <@%s>\n**原帖链接:** %s\n**安利内容:**\n%s", submissionID, i.Member.User.ID, recommendTitle, originalAuthor, originalURL, recommendContent),
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
